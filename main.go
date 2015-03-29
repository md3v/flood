package main

import "os"
import "log"
import "flag"
import "strconv"

var remote_host *string = flag.String("h", "", "remote host")
var remote_port *int = flag.Int("p", 3388, "remote port")
var bind_host *string = flag.String("H", "", "server bind host")
var bind_port *int = flag.Int("P", 3388, "server bind port")
var client *bool = flag.Bool("c", false, "start ctl mode")
var local *bool = flag.Bool("l", false, "connect local executor in ctl client mode")

func main() {
    flag.Parse()

    // setup logger
    log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

    connect_local := (*client && *remote_host == "") || !*client
    run_server := !*client

    stress := NewStress()

    flood := NewFlood()
    flood_rpc := NewFloodRpc(flood)
    flood.Register(flood_rpc)
    flood.Register(stress)

    if connect_local || *local {
        flood.ConnectLocal()
    }
    if *remote_host != "" {
        flood.Connect(*remote_host, strconv.Itoa(*remote_port), run_server)
    }
    if run_server {
        if *client {
            go flood.Serve(*bind_host, strconv.Itoa(*bind_port))
        } else {
            flood.Serve(*bind_host, strconv.Itoa(*bind_port))
        }
    }
    if *client {
        ServeCtl(os.Stdin, os.Stdout, flood_rpc)
    }
}
