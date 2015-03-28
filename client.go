package main

import "fmt"
import "time"
import "strings"
import "net/http"
import "crypto/tls"

const USER_AGENT = "flood"

type HttpArgs struct {
    method string
    url string
    body string
    headers string
    ssl_skip bool
}

func HttpClient(id int, out chan *Msg, a *FloodArgs) {
    args := a.Client_args.(*HttpArgs)

    tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: args.ssl_skip},
    }
    client := &http.Client{Transport: tr}


    for i := 0; i < a.Requests; i++ {
        msg := &Msg{Code: 0}
        req, err := http.NewRequest(args.method, args.url,
            strings.NewReader(args.body))
        if err != nil {
            fmt.Printf("%d/%d Failed creating request, error: %s\n",
                id, i, err)
            out <- msg
            continue
        }
        req.Header.Set("User-Agent", USER_AGENT)
        // TODO 
        if len(args.headers) > 0 {
            chunks := strings.SplitN(args.headers, ":", 2)
            req.Header.Add(chunks[0], chunks[1])
        }

        ts := time.Now()
        resp, err := client.Do(req)
        msg.Time = time.Now().Sub(ts)

        if err != nil {
            fmt.Printf("%d/%d Request failed, error: %s\n", id, i, err)
            out <- msg
            continue
        }

        msg.Code = resp.StatusCode

        out <- msg
    }

}
