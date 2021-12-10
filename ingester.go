package main

import (
	"database/sql"
	"fmt"
	logging "log"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go"
	wal "github.com/tidwall/wal"
)

type RequestReceived struct {
	Streams []struct {
		Stream map[string]string `json:"stream"`
		Values [][2]string `json:"values"`
	} `json:"streams"` 
}

type Queue struct {
	mutex sync.Mutex
	qu []RequestReceived
}

type WalStruct struct {
	l *wal.Log
	index uint64
}

var queue Queue
var w WalStruct
var delta1 float64
//var delta2 float64

const (
	layoutISO = "2006-01-02"
	walDirPath = "/home/log-ingester"
)

func main() {
	
	time.Sleep(time.Second*2)
	// Creation of the connection and setup database if not done before
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
			label Array(String),
			timestamp Date,
			msg String
		)
		ENGINE = MergeTree()
		ORDER BY timestamp
	`)
	if err != nil {
		logging.Fatal(err)
	}


	// Initialize goroutines:
	// 1º: send queue to database each X seconds
	// 2º: create api and keep serving connections while work is done

	go sendQueue(connect)
	createAPI()

}


func sendQueue(connect *sql.DB) {
	for {
		time.Sleep(time.Second*10)

		queue.mutex.Lock()
		if len(queue.qu) != 0 {
			start := time.Now()
			createInsert(connect)
			end := time.Now()
			delta1 = end.Sub(start).Seconds()
			logging.Println(delta1)
			queue.qu = nil	
		}
		queue.mutex.Unlock()
	}
}





func createInsert (connect *sql.DB) {
	logging.Println(queue.qu)
	tx, err := connect.Begin()
	if err != nil {
		logging.Fatal(err)
	}

	stmt, err := tx.Prepare("INSERT INTO logs.log (label, timestamp, msg) VALUES (?,?,?)")
	if err != nil {
		logging.Fatal(err)
	}

	var sliceLabel []string
	for q := 0; q<len(queue.qu); q++ {
		for i := 0; i<len(queue.qu[q].Streams); i++ {
			// This happens in every stream
			
			count := 0
			for k, e := range queue.qu[q].Streams[i].Stream {
				count++
				sliceLabel = append(sliceLabel, k + ":" + e)
			}

			for _, row := range queue.qu[q].Streams[i].Values {
				tim, _ := time.Parse(layoutISO, row[0])
				_, err = stmt.Exec(sliceLabel, tim, row[1])
				if err != nil {
					logging.Fatal(err)
				}
				logging.Println(sliceLabel)
				logging.Println(row[0])
				logging.Println(row[1])
			}

		}
	}
	
	err = tx.Commit()
		if err != nil {
			logging.Fatal(err)
		}
}