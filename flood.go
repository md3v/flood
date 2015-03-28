package main

import "fmt"
import "io"
import "net"
import "sync"
import "reflect"
import "net/rpc"
import "container/list"

type FloodRpcArgs struct {
    Service string
    Args map[string]string
}

type FloodRpcReply struct {
    Reply map[string]string
    Peers []FloodRpcReply
}

type FloodRpc struct {
    flood *Flood
}

func NewFloodRpc(f *Flood) *FloodRpc {
    r := &FloodRpc{
        flood: f,
    }
    return r
}

func (r *FloodRpc) Run(args FloodRpcArgs, reply *FloodRpcReply) error {
    var local_call *rpc.Call
    calls := list.New()

    if r.flood.local != nil {
        fmt.Printf("Calling local, service: %s\n", args.Service)
        local_call = r.flood.local.Go(args.Service, args, reply, nil)
        calls.PushBack(local_call)
    }

    r.flood.peers_lck.Lock()
    for e := r.flood.peers.Front(); e != nil; e = e.Next() {
        r := &FloodRpcReply{}
		call := e.Value.(*rpc.Client).Go("FloodRpc.Run", args, r, nil)
        calls.PushFront(call)
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

type Flood struct {
    client_conns map[string]net.Conn
    server_conns map[string]net.Conn

    local *rpc.Client
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

func (f *Flood) addPeer(conn io.ReadWriteCloser) {
    peer := rpc.NewClient(conn)

    f.peers_lck.Lock()
    f.peers.PushFront(peer)
    f.peers_lck.Unlock()
}

func (f *Flood) ConnectLocal() {
    conn1, conn2 := net.Pipe()
    f.local = rpc.NewClient(conn1)
    go f.rpc_server.ServeConn(conn2)
    fmt.Printf("Connected to local rpc\n")
}

func (f *Flood) Connect(host string, port string, server bool) {
    // client connection
    conn, err := net.Dial("tcp", host + ":" + port)
    if err != nil {
        fmt.Printf("Failed dialing, host: %s, port: %s, err: %s",
            host, port, err)
    }
    // add client connection
    f.addPeer(conn)

    if server {
        // server connection
        conn, err = net.Dial("tcp", host + ":" + port)
        if err != nil {
            fmt.Printf("Failed dialing, host: %s, port: %s, err: %s",
                host, port, err)
        }
        go f.rpc_server.ServeConn(conn)
    }
}

func (f *Flood) Register(rcvr interface{}) {
    err := f.rpc_server.Register(rcvr)
    if err != nil {
        fmt.Printf("Failed registering, rcvr: %s, err: %s", rcvr, err)
    }
}

func (f *Flood) Serve(host string, port string) {
    l, err := net.Listen("tcp", host + ":" + port)
	if err != nil {
        fmt.Printf("Failed listening, host: %s, port: %s, err: %s",
            host, port, err)
        return
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
        if err != nil {
            fmt.Printf("Failed accepting, host: %s, port: %s, err: %s",
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
