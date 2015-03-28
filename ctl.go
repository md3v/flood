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
        line, err := reader.ReadLine()
        if err == io.EOF {
            fmt.Printf("Failed reading, error: %s\n", err)
            break
        }

        fmt.Printf("line: %s\n", line)
        service, args := parseLine(line)

        flood_args := FloodRpcArgs{
            Service: service + ".Run",
            Args: args,
        }

        flood_reply := &FloodRpcReply{}

        flood_rpc.Run(flood_args, flood_reply)
        for key, value := range flood_reply.Reply {
            fmt.Println("Key:", key, "Value:", value)
        }
    }
}

// Stress stress_type=http stress_concurrency=1 stress_iterations=1 http_method=GET http_url=http://example.com
func parseLine(line string) (string, map[string]string) {
    chunks := strings.Split(line, " ")
    service := chunks[0]
    args := make(map[string]string)
    for _, chunk := range chunks[1:] {
        c := strings.SplitN(chunk, "=", 2)
        args[c[0]] = c[1]
    }

    return service, args
}

func readBody(body_param string) string {
    body := body_param
    if len(body_param) > 0 && body_param[0] == '@' {
        buf, err := ioutil.ReadFile(body_param[1:])
        if err != nil {
            fmt.Printf("Failed reading file, error: %s\n", err)
            return body_param
        }
        body = string(buf)
    }

    return body
}
