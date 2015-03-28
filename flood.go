package main

import "fmt"
import "io"
import "net"
import "sync"
import "net/rpc"
import "container/list"

type Flood struct {
    client_conns map[string]net.Conn
    server_conns map[string]net.Conn

    peers *list.List
    peers_lck *sync.Mutex

    rpc_server *rpc.Server
}

type FloodReply struct {
    reply map[string]string
    peers []FloodReply
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
    go f.rpc_server.ServeConn(conn1)
    go f.addPeer(conn2)
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
    }
}

func (f *Flood) Register(rcvr interface{}, local bool) {
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
