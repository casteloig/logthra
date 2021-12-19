package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	logging "log"
	"net/http"
	"time"

	model "log_tool/model"
	pb "log_tool/model/proto/log"

	"google.golang.org/grpc"
)


func pushHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("asked for push: ", r.Method)
		if r.Method == "POST" {

			var logs model.RequestReceived
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				logging.Fatal(err)
			}
			err = json.Unmarshal(body, &logs)
			if err != nil {
				logging.Fatal(err)
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
			
			conn, err := grpc.Dial("ingester:9011", grpc.WithInsecure())
			if err != nil {
				logging.Fatal(err)
			}
			defer conn.Close()

			c := pb.NewIngesterServiceClient(conn)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			
			response, err := c.PushToIngester(ctx, r)
			if err != nil {
				logging.Fatal(err)
			}

			logging.Println(response)

		} else {
			http.Error(w, "Only POST methods are supported", http.StatusNotFound)
		}
	})
}



func main() {
	http.Handle("/api/push", pushHandler())

	err := http.ListenAndServe(":9010", nil)
	logging.Println("Listening :9010")
	if err != nil {
		logging.Fatal(err)
	}
}
