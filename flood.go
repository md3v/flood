package main

import "log"
import "os"
import "net"
import "sync"
import "strconv"
import "reflect"
import "net/rpc"
import "container/list"

type FloodRpcReq struct {
    Source string
    Service string
    Args map[string]string
}

type FloodRpcReply struct {
    Service string
    NodeName string
    Reply map[string]string
    Peers []FloodRpcReply
}

type FloodRpc struct {
    flood *Flood
    name string
}

func NewFloodRpc(f *Flood) *FloodRpc {
    r := &FloodRpc{
        flood: f,
    }
    name, err := os.Hostname()
    if err != nil {
        log.Printf("Error getting hostname, err: %s", err)
    }
    r.name = name + "#" + strconv.Itoa(os.Getpid())
    return r
}

func (r *FloodRpc) Run(req FloodRpcReq, reply *FloodRpcReply) error {
    log.Printf("FloodRpc.Run/%s, src: %s", req.Service, req.Source)

    var local_call *rpc.Call
    calls := list.New()

    reply.NodeName = r.name

    if r.flood.local != nil {
        reply.Service = req.Service
        local_call = r.flood.local.rpc.Go(req.Service, req, reply, nil)
        calls.PushBack(local_call)
    }

    orig_src := req.Source

    r.flood.peers_lck.Lock()
    for e := r.flood.peers.Front(); e != nil; e = e.Next() {
        client := e.Value.(*floodPeer)
        // check to eliminate network loops
        if client.addr_dst != orig_src {
            log.Printf("Forward FloodRpc.Run/%s, src: %s, dst: %s",
                req.Service, orig_src, client.addr_dst)
            req.Source = client.addr_src
            rep := &FloodRpcReply{}
            call := client.rpc.Go("FloodRpc.Run", req, rep, nil)
            calls.PushFront(call)
        }
	}
    r.flood.peers_lck.Unlock()

    cases := make([]reflect.SelectCase, calls.Len())
    i := 0
    for e := calls.Front(); e != nil; e = e.Next() {
        call := e.Value.(*rpc.Call)
        cases[i] = reflect.SelectCase{
            Dir: reflect.SelectRecv,
            Chan: reflect.ValueOf(call.Done)}
        i++
    }

    reply.Peers = make([]FloodRpcReply, calls.Len())
    i = 0
    for i < calls.Len() {
        _, value, _ := reflect.Select(cases)
        call := value.Interface().(*rpc.Call)

        if call != local_call {
            r := call.Reply.(*FloodRpcReply)
            reply.Peers = append(reply.Peers, *r)
        }
        i++
    }

    return nil
}

type floodPeer struct {
    addr_src string
    addr_dst string
    conn net.Conn
    rpc *rpc.Client
}

type Flood struct {
    client_conns map[string]net.Conn
    server_conns map[string]net.Conn

    local *floodPeer
    peers *list.List
    peers_lck *sync.Mutex

    rpc_server *rpc.Server
}

func NewFlood() *Flood {
    f := &Flood{
        client_conns: make(map[string]net.Conn),
        server_conns: make(map[string]net.Conn),
        peers: list.New(),
        peers_lck: &sync.Mutex{},
        rpc_server: rpc.NewServer(),
    }
    return f
}

func (f *Flood) createPeer(conn net.Conn) *floodPeer {
    src, _, _ := net.SplitHostPort(conn.LocalAddr().String())
    dst, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

    peer := &floodPeer{
        addr_src: src,
        addr_dst: dst,
        conn: conn,
        rpc: rpc.NewClient(conn),
    }

    return peer
}

func (f *Flood) addPeer(conn net.Conn) {
    peer := f.createPeer(conn)

    f.peers_lck.Lock()
    f.peers.PushFront(peer)
    f.peers_lck.Unlock()
}

func (f *Flood) ConnectLocal() {
    conn1, conn2 := net.Pipe()
    f.local = f.createPeer(conn1)
    go f.rpc_server.ServeConn(conn2)
    log.Printf("Connected to local rpc\n")
}

func (f *Flood) Connect(host string, port string, server bool) error {
    // client connection (me/client -> rpc server)
    conn, err := net.Dial("tcp", host + ":" + port)
    if err != nil {
        log.Printf("Failed dialing, host: %s, port: %s, err: %s",
            host, port, err)
        return err
    }
    // add client connection
    f.addPeer(conn)

    if server {
        // server connection (peer -> rpc server/me)
        conn, err = net.Dial("tcp", host + ":" + port)
        if err != nil {
            log.Printf("Failed dialing, host: %s, port: %s, err: %s",
                host, port, err)
            return err
        }
        go f.rpc_server.ServeConn(conn)
    }

    return nil
}

func (f *Flood) Register(rcvr interface{}) {
    err := f.rpc_server.Register(rcvr)
    if err != nil {
        log.Printf("Failed registering, rcvr: %s, err: %s", rcvr, err)
    }
}

func (f *Flood) Serve(host string, port string) {
    l, err := net.Listen("tcp", host + ":" + port)
	if err != nil {
        log.Printf("Failed listening, host: %s, port: %s, err: %s",
            host, port, err)
        return
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
        if err != nil {
            log.Printf("Failed accepting, host: %s, port: %s, err: %s",
                host, port, err)
            continue
        }

        host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
        _, ok := f.server_conns[host]
        if !ok {
            f.server_conns[host] = conn
            go f.rpc_server.ServeConn(conn)
        } else {
            f.client_conns[host] = conn
            go f.addPeer(conn)
        }
	}

}
