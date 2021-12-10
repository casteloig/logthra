package main

import (
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

			fmt.Println(logs)
			queue.mutex.Lock()
			queue.qu = append(queue.qu, logs)
			queue.mutex.Unlock()
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