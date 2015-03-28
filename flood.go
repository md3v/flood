package main

import "time"

type FloodArgs struct {
    Clients int
    Requests int
    Client_name string
    Client_args interface{}
}

type FloodReply struct {
    Success int
    Fail int
    Min_time time.Duration
    Max_time time.Duration
    Avg_time time.Duration
    Total_time time.Duration
}

type Msg struct {
    Code int
    Time time.Duration
}

type Flood struct {
    client_func map[string]interface{}
}

func NewFlood() *Flood {
    f := &Flood{
        client_func: make(map[string]interface{}),
    }
    f.client_func["http"] = HttpClient
    return f
}

func (f *Flood) Run(args *FloodArgs, reply *FloodReply) error {
    v := f.client_func[args.Client_name]
    client_func := v.(func(int, chan *Msg, *FloodArgs))

    total_requests := args.Clients * args.Requests
    out := make(chan *Msg, total_requests)

    ts := time.Now()
    for i := 0; i < args.Clients; i++ {
        go client_func(i, out, args)
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

    reply.Success = success
    reply.Fail = fail
    reply.Min_time = min_time
    reply.Max_time = max_time
    reply.Avg_time = avg_time
    reply.Total_time = total_time

    return nil
}
