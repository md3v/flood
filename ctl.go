package main

import "fmt"
import "io"
import "bufio"
import "strings"
import "net/textproto"
import "io/ioutil"

func ServeCtl(in io.Reader, out io.Writer, flood_rpc *FloodRpc) {
    reader := textproto.NewReader(bufio.NewReader(in))
    for {
        fmt.Fprint(out, "> ")
        line, err := reader.ReadLine()
        if err == io.EOF {
            fmt.Printf("Failed reading, error: %s\n", err)
            break
        }

        service, args := parseLine(line)

        flood_args := FloodRpcArgs{
            Service: service,
            Args: args,
        }

        flood_reply := &FloodRpcReply{}

        flood_rpc.Run(flood_args, flood_reply)

        fmt.Fprint(out, service)
        for key, value := range flood_reply.Reply {
            fmt.Fprintf(out, ", %s: %s", key, value)
        }
        fmt.Fprintf(out, "\n")
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
            fmt.Printf("Failed reading file, error: %s\n", err)
        } else {
            out = string(buf)
        }
    }

    return out
}
