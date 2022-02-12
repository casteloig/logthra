package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	logging "log"
	"net/http"
	"os"
	"time"

	model "log_tool/model"
	pb "log_tool/model/proto/log"

	"google.golang.org/grpc"
)

var limiter = NewIPRateLimiter()

func pushHandler(conn *grpc.ClientConn) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("asked for push: ", r.Method)
		if r.Method == "POST" {

			var logs model.RequestReceived
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				logging.Fatal(err)
				http.Error(w, "Bad body message", http.StatusBadRequest)
			}
			err = json.Unmarshal(body, &logs)
			if err != nil {
				logging.Fatal(err)
				http.Error(w, "Bad body message", http.StatusBadRequest)
			}

			tenantID := r.Header.Get("tenant-id")
			if len(tenantID) != 10 {
				logging.Println("tenant-id invalid (has to be lenght 10)")
				http.Error(w, "Header is not correct", http.StatusBadRequest)
				return
			}
			logs.TenantID = tenantID

			r := &pb.RequestReceivedProto{}
			r.TenantID = logs.TenantID
			
			for _, a := range logs.Streams{
				strms := &pb.Strms{}
				strms.Stream = a.Stream
				for _, b := range a.Values {
					odv := &pb.OneDimValue{
						ValTime: b[0],
						ValMsg: b[1],
					}
					strms.TwoDimValue = append(strms.TwoDimValue, odv)
				}
				r.Streams = append(r.Streams, strms)
			}
			

			c := pb.NewIngesterServiceClient(conn)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			
			response, err := c.PushToIngester(ctx, r)
			if err != nil {
				logging.Fatal(err)
				http.Error(w, "Header is not correct", http.StatusServiceUnavailable)
			}

			logging.Println(response)

		} else {
			http.Error(w, "Only POST methods are supported", http.StatusNotFound)
		}
	})
}

func health (w http.ResponseWriter, r *http.Request){
	w.WriteHeader(http.StatusOK)
}


func main() {

	url := os.Getenv("CONFIG_INGESTER_URL")
	conn, err := grpc.Dial(url + ":9011", grpc.WithInsecure())
	if err != nil {
		logging.Fatal(err)
	}
	defer conn.Close()

	mux := http.NewServeMux()
	mux.Handle("/api/push", pushHandler(conn))
	mux.HandleFunc("/api/health", health)

	err = http.ListenAndServe(":9010", limitMiddleware(mux))
	logging.Println("Listening :9010")
	if err != nil {
		logging.Fatal(err)
	}

}


func limitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        limiter := limiter.GetLimiter(r.Header.Get("tenant-id"))
        if !limiter.Allow() {
            http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
            return
        }

        next.ServeHTTP(w, r)
    })
}