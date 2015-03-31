package main

import "crypto/tls"
import "io"
import "io/ioutil"
import "log"
import "net/http"
import "strconv"
import "strings"
import "time"

type stressMsg struct {
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

// Stress.Run concurrency=5 iterations=10 type=http http_method=GET http_url=http://example.com
func (f *Stress) Run(req FloodRpcReq, reply *FloodRpcReply) error {
    args := req.Args
    concurrency, _ := strconv.Atoi(args["concurrency"])
    iterations, _ := strconv.Atoi(args["iterations"])

    log.Printf("%s, concurrency: %d, iterations: %d",
        req.Service, concurrency, iterations)

    v := f.stress_func[args["type"]]
    stress := v.(func(int, chan *stressMsg, map[string]string))

    total_requests := concurrency * iterations
    out := make(chan *stressMsg, total_requests)

    ts := time.Now()
    for i := 0; i < concurrency; i++ {
        go stress(i, out, args)
    }

    count := 0
    success := 0
    fail := 0
    errors := 0
    time_count := 0
    min_time := 100 * time.Hour
    max_time := time.Duration(0)
    avg_time := time.Duration(0)
    for count < total_requests {
        rsp := <-out
        count++;
        if rsp.Code == 200 {
            success++
        } else if rsp.Code == -1 {
            errors++
            // errors doesn't impact timing
            continue
        } else {
            fail++
        }
        time_count++
        if rsp.Time < min_time {
            min_time = rsp.Time
        }
        if rsp.Time > max_time {
            max_time = rsp.Time
        }
        avg_time = (avg_time * time.Duration(time_count - 1)  + rsp.Time) / time.Duration(time_count)
    }
    min_time /= time.Millisecond
    max_time /= time.Millisecond
    avg_time /= time.Millisecond
    total_time := time.Now().Sub(ts) / time.Millisecond

    reply.Reply = make(map[string]string)
    reply.Reply["success"] = strconv.Itoa(success)
    reply.Reply["fail"] = strconv.Itoa(fail)
    reply.Reply["errors"] = strconv.Itoa(errors)
    reply.Reply["min_time"] = strconv.Itoa(int(min_time))
    reply.Reply["max_time"] = strconv.Itoa(int(max_time))
    reply.Reply["avg_time"] = strconv.Itoa(int(avg_time))
    reply.Reply["total_time"] = strconv.Itoa(int(total_time))

    return nil
}

const USER_AGENT = "flood"

func HttpTest(id int, out chan *stressMsg, args map[string]string) {
    ssl_skip := "true" == args["http_ssl_skip"]
    disable_keep_alive := "true" == args["http_disable_keepalive"]
    iterations, _ := strconv.Atoi(args["iterations"])

    tr := &http.Transport{
        TLSClientConfig: &tls.Config{
            InsecureSkipVerify: ssl_skip,
        },
        // DisableKeepAlives, if true, prevents re-use of TCP connections
        // between different HTTP requests.
        DisableKeepAlives: disable_keep_alive,
    }
    client := &http.Client{Transport: tr}

    for i := 0; i < iterations; i++ {
        msg := &stressMsg{Code: -1, Time: 0}
        req, err := http.NewRequest(args["http_method"], args["http_url"],
            strings.NewReader(args["http_body"]))
        if err != nil {
            log.Printf("%d/%d Failed creating request, error: %s\n",
                id, i, err)
            out <- msg
            continue
        }
        req.Header.Set("User-Agent", USER_AGENT)
        // TODO parse multiple headers
        if len(args["http_headers"]) > 0 {
            chunks := strings.SplitN(args["http_headers"], ":", 2)
            req.Header.Add(chunks[0], chunks[1])
        }

        ts := time.Now()

        res, err := client.Do(req)
        if err != nil {
            msg.Time = time.Now().Sub(ts)
            log.Printf("%d/%d Request failed, error: %s\n", id, i, err)
        } else {
            io.Copy(ioutil.Discard, res.Body)
            res.Body.Close()
            msg.Time = time.Now().Sub(ts)
            msg.Code = res.StatusCode
        }

        out <- msg
    }

}
