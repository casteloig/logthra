package main

import (
	"database/sql"
	"fmt"
	logging "log"

	"github.com/ClickHouse/clickhouse-go"
)

type BatchReceived struct {
	Streams []struct {
		Stream map[string]string `json:"stream"`
		Values [][2]string `json:"values"`
	} `json:"streams"` 
}


func main() {

	connect, err := sql.Open("clickhouse", "tcp://clickhouse:9000?debug=true")
	
	if err != nil {
		logging.Fatal(err)
	}

	if err := connect.Ping(); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			fmt.Printf("[%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		} else {
			fmt.Println(err)
		}
		return
	}

	_, err = connect.Exec("CREATE DATABASE IF NOT EXISTS logs")
	if err != nil {
		logging.Fatal(err)
	}

	_, err = connect.Exec(`
		CREATE TABLE IF NOT EXISTS logs.log(
			label Map(String, String),
			timestamp Date,
			msg String
		)
		ENGINE = MergeTree()
		ORDER BY timestamp
	`)
	if err != nil {
		logging.Fatal(err)
	}

	createAPI(connect)
}

func createInsert (connect *sql.DB, logs BatchReceived) {
	tx, err := connect.Begin()
	if err != nil {
		logging.Fatal(err)
	}

	var stmString string
	for i := 0; i<len(logs.Streams); i++ {
		// This happens in every stream
		stmString = "INSERT INTO logs.log SELECT map("

		// We group all labels of the stream
		count := 0
		for k, e := range logs.Streams[i].Stream {
			count++
			stmString += "'" + k + "'" + "," + "'" + e + "'"
			if count < len(logs.Streams[i].Stream) {
				stmString += ","
			} else if count == len(logs.Streams[i].Stream) {
				stmString += "),"
			}
		}

		for _, row := range logs.Streams[i].Values {
			stmWorking := stmString
			stmWorking += row[0] + "," + "'" + row[1] + "'"
			
			statement, err := tx.Prepare(stmWorking)
			if err != nil {
				logging.Fatal(err)
			}
			_, err = statement.Exec()
			if err != nil {
				logging.Fatal(err)
			}

			fmt.Println(stmWorking)
		}
	}
	err = tx.Commit()
		if err != nil {
			logging.Fatal(err)
		}
}