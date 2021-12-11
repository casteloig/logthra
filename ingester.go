package main

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"fmt"
	logging "log"
	"os"
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
	myWal WalStruct
}

type WalStruct struct {
	l *wal.Log
}

var queue Queue

const (
	layoutISO = "2006-01-02"
	walDirPath = "/home/log-ingester/"
)

func main() {

	var err error

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


	queue.myWal.l, err = wal.Open(walDirPath, nil)
	if err != nil {
		logging.Fatal(err)
	}
	defer queue.myWal.l.Close()

	queue.mutex.Lock()
	err = recoverWal()
	if err != nil {
		logging.Fatal(err)
	}
	queue.mutex.Unlock()

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
			createInsert(connect)
			queue.qu = nil
			lastIndex, err := queue.myWal.l.LastIndex()
			if err != nil {
				logging.Fatal(err)
			} else if lastIndex != 0{
				queue.myWal.l.Close()
				os.RemoveAll(walDirPath)
				queue.myWal.l, err = wal.Open(walDirPath, nil)
				if err != nil {
					logging.Fatal(err)
				}
			}
		}
		queue.mutex.Unlock()





		lastIndex, err := queue.myWal.l.LastIndex()
		if err != nil {
			logging.Fatal(err)
		} else if lastIndex != 0{
			logging.Println("Despues de mandar queue nos encontramos con...")
			for i := 1; i <= int(lastIndex); i++ {
				data, _ := queue.myWal.l.Read(uint64(i))
				queue.qu = append(queue.qu, decodeToRequestReceived(data))
				logging.Println(decodeToRequestReceived(data))
			} 
		}



		
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
		sliceLabel = nil
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


func recoverWal() error {
	lastIndex, err := queue.myWal.l.LastIndex()
	if err != nil {
		return err
	} else if lastIndex != 0{
		for i := 1; i <= int(lastIndex); i++ {
			data, err := queue.myWal.l.Read(uint64(i))
			logging.Println(data)
			if err != nil {
				return err
			}
			queue.qu = append(queue.qu, decodeToRequestReceived(data))
			logging.Println(decodeToRequestReceived(data))
		} 
	}

	return nil
}


func decodeToRequestReceived(s []byte) RequestReceived {

	r := RequestReceived{}
	dec := gob.NewDecoder(bytes.NewReader(s))
	err := dec.Decode(&r)
	if err != nil {
		logging.Fatal(err)
	}
	return r
}