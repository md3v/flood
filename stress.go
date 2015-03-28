package main

import "fmt"
import "time"
import "strconv"
import "strings"
import "net/http"
import "crypto/tls"

type Msg struct {
    Code int
    Time time.Duration
}

type Stress struct {
    stress_func map[string]interface{}
}

func NewStress() *Stress {
    f := &Stress{
        stress_func: make(map[string]interface{}),
    }
    f.stress_func["http"] = HttpTest
    return f
}

func (f *Stress) Run(a FloodRpcArgs, reply *FloodRpcReply) error {
    fmt.Printf("Stress\n")
    args := a.Args
    concurrency, _ := strconv.Atoi(args["stress_concurrency"])
    iterations, _ := strconv.Atoi(args["stress_iterations"])

    v := f.stress_func[args["stress_type"]]
    stress := v.(func(int, chan *Msg, map[string]string))

    total_requests := concurrency * iterations
    out := make(chan *Msg, total_requests)

    ts := time.Now()
    for i := 0; i < concurrency; i++ {
        go stress(i, out, args)
    }

    count := 0
    success := 0
    fail := 0
    min_time := 100 * time.Hour
    max_time := time.Duration(0)
    avg_time := time.Duration(0)
    for count < total_requests {
        rsp := <-out
        count++;
        if rsp.Code == 200 {
            success++
        } else {
            fail++
        }
        if rsp.Time < min_time {
            min_time = rsp.Time
        }
        if rsp.Time > max_time {
            max_time = rsp.Time
        }
        avg_time = (avg_time * time.Duration(count - 1)  + rsp.Time) / time.Duration(count)
    }
    min_time /= time.Millisecond
    max_time /= time.Millisecond
    avg_time /= time.Millisecond
    total_time := time.Now().Sub(ts) / time.Millisecond

    reply.Reply = make(map[string]string)
    reply.Reply["stress_success"] = strconv.Itoa(success)
    reply.Reply["stress_fail"] = strconv.Itoa(fail)
    reply.Reply["stress_min_time"] = strconv.Itoa(int(min_time))
    reply.Reply["stress_max_time"] = strconv.Itoa(int(max_time))
    reply.Reply["stress_avg_time"] = strconv.Itoa(int(avg_time))
    reply.Reply["stress_total_time"] = strconv.Itoa(int(total_time))

    return nil
}

const USER_AGENT = "flood"

func HttpTest(id int, out chan *Msg, args map[string]string) {
    ssl_skip := "true" == args["http_ssl_skip"]
    iterations, _ := strconv.Atoi(args["stress_iterations"])

    tr := &http.Transport{
        TLSClientConfig: &tls.Config{
            InsecureSkipVerify: ssl_skip,
        },
    }
    client := &http.Client{Transport: tr}

    for i := 0; i < iterations; i++ {
        msg := &Msg{Code: 0}
        req, err := http.NewRequest(args["http_method"], args["http_url"],
            strings.NewReader(args["http_body"]))
        if err != nil {
            fmt.Printf("%d/%d Failed creating request, error: %s\n",
                id, i, err)
            out <- msg
            continue
        }
        req.Header.Set("User-Agent", USER_AGENT)
        // TODO 
        if len(args["http_headers"]) > 0 {
            chunks := strings.SplitN(args["http_headers"], ":", 2)
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
