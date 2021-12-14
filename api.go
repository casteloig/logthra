package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	logging "log"
	"net/http"
)


func pushHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("asked for push: ", r.Method)
		if r.Method == "POST" {
	
			logging.Println("inside post/push")
	
			var logs RequestReceived
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

			buf := encodeToBytes(logs)
			queue.mutex.Lock()
			lastIndex, err := queue.myWal.l.LastIndex()
			if err != nil {
				logging.Fatal(err)
			}
			err = queue.myWal.l.Write(lastIndex+1, buf)
			if err != nil {
				logging.Fatal(err)
			}
			
			logging.Println(logs)
			queue.qu = append(queue.qu, logs)
			queue.mutex.Unlock()
		} else {
			http.Error(w, "Only POST methods are supported", http.StatusNotFound)
		}
	})
}



func createAPI() {
	http.Handle("/api/push", pushHandler())

	err := http.ListenAndServe(":9010", nil)
	logging.Println("Listening :9010")
	if err != nil {
		logging.Fatal(err)
	}
}


func encodeToBytes(p interface{}) []byte  {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(p)
	if err != nil {
		logging.Fatal(err)
	}
	
	return buf.Bytes()
}