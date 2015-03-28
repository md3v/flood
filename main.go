package main

import "fmt"
import "flag"
import "strconv"
import "io/ioutil"

var clients *int = flag.Int("c", 1, "Number of clients to simulate")
var requests *int = flag.Int("n", 1, "Number of requests per client")
var http_method *string = flag.String("X", "GET", "http method")
var http_url *string = flag.String("u", "http://example.com", "http url")
var http_body *string = flag.String("d", "", "http body")
var http_headers *string = flag.String("H", "", "http headers")
var ssl_skip *bool = flag.Bool("k", false, "skip ssl cert verification")

var remote_host *string = flag.String("h", "localhost", "remote host")
var remote_port *int = flag.Int("p", 3388, "remote port")
var bind_host *string = flag.String("H", "0.0.0.0", "server bind host")
var bind_port *int = flag.Int("P", 3388, "server bind port")
var client *bool = flag.Bool("c", false, "start in client ctl (no server, no local executor)")

func main() {
    flag.Parse()

    total := *clients * *requests

    body := *http_body
    if len(*http_body) > 0 && (*http_body)[0] == '@' {
        buf, err := ioutil.ReadFile((*http_body)[1:])
        if err != nil {
            fmt.Printf("Failed reading file, error: %s\n", err)
            return
        }
        body = string(buf)
    } else {
        body = *http_body
    }

    fmt.Printf("clients: %d, requests per client: %d, total requests: %d\n",
        *clients, *requests, total)

    args := map[string]string{
        "http_method": *http_method,
        "http_url": *http_url,
        "http_body": body,
        "http_headers": *http_headers,
        "http_ssl_skip": strconv.FormatBool(*ssl_skip),
        "stress_concurrency": strconv.Itoa(*clients),
        "stress_iterations": strconv.Itoa(*requests),
        "stress_type": "http",
    }

    reply := &FloodReply{
        reply: make(map[string]string),
        peers: nil,
    }

    stress := NewStress()
    stress.Run(args, reply)

    fmt.Printf("success: %d, fail: %d, min: %dms, max: %dms, avg: %dms, total test time: %dms\n",
        reply.reply["stress_success"], reply.reply["stress_fail"], reply.reply["stress_min_time"],
        reply.reply["stress_max_time"], reply.reply["stress_avg_time"], reply.reply["stress_total_time"])
}
