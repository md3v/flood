package main

import "bufio"
import "fmt"
import "io"
import "io/ioutil"
import "log"
import "net/textproto"
import "sort"
import "strings"

func ServeCtl(in io.Reader, out io.Writer, flood_rpc *FloodRpc) {
    reader := textproto.NewReader(bufio.NewReader(in))
    for {
        fmt.Fprint(out, "> ")
        line, err := reader.ReadLine()
        if err == io.EOF {
            log.Printf("Failed reading, error: %s\n", err)
            break
        }

        service, args := parseLine(line)

        flood_req := FloodRpcReq{
            Service: service,
            Args: args,
        }

        flood_reply := &FloodRpcReply{}
        flood_rpc.Run(flood_req, flood_reply)
        printReply(out, flood_reply)
    }
}

func printReply(out io.Writer, reply *FloodRpcReply) {
    if reply.Service != "" {
        fmt.Fprintf(out, "[%s] %s", reply.NodeName, reply.Service)
        var keys []string
        for key := range reply.Reply {
            keys = append(keys, key)
        }
        sort.Strings(keys)
        for _, key := range keys {
            fmt.Fprintf(out, ", %s: %s", key, reply.Reply[key])
        }
        fmt.Fprintf(out, "\n")
    }
    for _, peer_reply := range reply.Peers {
        printReply(out, &peer_reply)
    }
}

// Stress.Run concurrency=5 iterations=10 type=http http_method=GET http_url=http://example.com
func parseLine(line string) (string, map[string]string) {
    chunks := strings.Split(line, " ")
    service := chunks[0]
    args := make(map[string]string)
    for _, chunk := range chunks[1:] {
        c := strings.SplitN(chunk, "=", 2)
        args[c[0]] = tryReadFile(c[1])
    }

    return service, args
}

func tryReadFile(param string) string {
    out := param
    if len(param) > 0 && param[0] == '@' {
        buf, err := ioutil.ReadFile(param[1:])
        if err != nil {
            log.Printf("Failed reading file, error: %s\n", err)
        } else {
            out = string(buf)
        }
    }

    return out
}
