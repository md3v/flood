package main

import "os"
import "flag"
import "strconv"

/* var clients *int = flag.Int("c", 1, "Number of clients to simulate") */
/* var requests *int = flag.Int("n", 1, "Number of requests per client") */
/* var http_method *string = flag.String("X", "GET", "http method") */
/* var http_url *string = flag.String("u", "http://example.com", "http url") */
/* var http_body *string = flag.String("d", "", "http body") */
/* var http_headers *string = flag.String("H", "", "http headers") */
/* var ssl_skip *bool = flag.Bool("k", false, "skip ssl cert verification") */

var remote_host *string = flag.String("h", "", "remote host")
var remote_port *int = flag.Int("p", 3388, "remote port")
var bind_host *string = flag.String("H", "", "server bind host")
var bind_port *int = flag.Int("P", 3388, "server bind port")
var client *bool = flag.Bool("c", false, "start ctl mode")
var server *bool = flag.Bool("s", false, "start server in ctl mode")

func main() {
    flag.Parse()
    connect_local := (*client && *remote_host == "") || !*client
    run_server := !*client || *server

    stress := NewStress()

    flood := NewFlood()
    flood_rpc := NewFloodRpc(flood)
    flood.Register(flood_rpc)
    flood.Register(stress)

    if connect_local {
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

    /* total := *clients * *requests */

    /* fmt.Printf("clients: %d, requests per client: %d, total requests: %d\n", */
    /*     *clients, *requests, total) */

    /* args := map[string]string{ */
    /*     "http_method": *http_method, */
    /*     "http_url": *http_url, */
    /*     "http_body": body, */
    /*     "http_headers": *http_headers, */
    /*     "http_ssl_skip": strconv.FormatBool(*ssl_skip), */
    /*     "stress_concurrency": strconv.Itoa(*clients), */
    /*     "stress_iterations": strconv.Itoa(*requests), */
    /*     "stress_type": "http", */
    /* } */

    /* reply := &FloodReply{ */
    /*     reply: make(map[string]string), */
    /*     peers: nil, */
    /* } */

    /* stress := NewStress() */
    /* stress.Run(args, reply) */

    /* fmt.Printf("success: %s, fail: %s, min: %sms, max: %sms, avg: %sms, total test time: %sms\n", */
    /*     reply.reply["stress_success"], reply.reply["stress_fail"], reply.reply["stress_min_time"], */
    /*     reply.reply["stress_max_time"], reply.reply["stress_avg_time"], reply.reply["stress_total_time"]) */
}
