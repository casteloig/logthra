package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	logging "log"
	"net/http"
)


func pushHandler(connect *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("asked for push: ", r.Method)
		if r.Method == "POST" {
	
			logging.Println("inside post/push")
	
			var logs BatchReceived
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				logging.Fatal(err)
			}
			err = json.Unmarshal(body, &logs)
			if err != nil {
				logging.Fatal(err)
			}
			fmt.Println(logs)
			createInsert(connect, logs)
		}
	})
}



func createAPI(connect *sql.DB) {
	http.Handle("/api/push", pushHandler(connect))

	err := http.ListenAndServe(":9010", nil)
	logging.Println("Listening :9010")
	if err != nil {
		logging.Fatal(err)
	}
}