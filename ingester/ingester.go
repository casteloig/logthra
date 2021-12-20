package main

import (
	"context"
	"database/sql"
	"fmt"
	logging "log"
	"net"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/ClickHouse/clickhouse-go"
	wal "github.com/tidwall/wal"

	model "log_tool/model"
	pb "log_tool/model/proto/log"
)


type Queue struct {
	mutex sync.Mutex
	qu []model.RequestReceived
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

type server struct {
	pb.UnimplementedIngesterServiceServer
}

func (s *server) PushToIngester(ctx context.Context, message *pb.RequestReceivedProto) (*pb.Response, error) {
	rr := model.RequestReceived{}
	rr.TenantID = message.GetTenantID()
	for _, strms := range message.GetStreams() {
		var newValues [][2]string
		var nv [2]string
		for _, odv := range strms.TwoDimValue {
			nv[0] = odv.ValTime
			nv[1] = odv.ValMsg
			newValues = append(newValues, nv)
		}
		
		rr.Streams = append(rr.Streams, model.StreamStruct{Stream: strms.GetStream(), Values: newValues})
	}
	
	logging.Println("new request:")
	logging.Println(rr)
	queue.mutex.Lock()
	queue.qu = append(queue.qu, rr)
	queue.mutex.Unlock()

	return &pb.Response{Result: "accepted"}, nil
}


func main() {

	var err error

	// Create gRPC server and listen to connections
	go createGRPCServer()
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
			tenant_id FixedString(10),
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

	// Restore Wal once at the begining of the ingester
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

	sendQueue(connect)

}


// Empty the queue and create all inserts
func sendQueue(connect *sql.DB) {
	ticker := time.NewTicker(30 * time.Second)
	logging.Println("antes")
	for range ticker.C {
		logging.Println("ticking...")
		// TODO: REPLACE SLEEP FOR TICKERS 
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

	}
}


func createInsert(connect *sql.DB) {
	tx, err := connect.Begin()
	if err != nil {
		logging.Fatal(err)
	}

	stmt, err := tx.Prepare("INSERT INTO logs.log (tenant_id, label, timestamp, msg) VALUES (?,?,?,?)")
	if err != nil {
		logging.Fatal(err)
	}

	var sliceLabel []string
	for q := 0; q<len(queue.qu); q++ {
		for i := 0; i<len(queue.qu[q].Streams); i++ {
			// This happens in every stream
			sliceLabel = nil
			count := 0
			for k, e := range queue.qu[q].Streams[i].Stream {
				count++
				sliceLabel = append(sliceLabel, k + ":" + e)
			}

			for _, row := range queue.qu[q].Streams[i].Values {
				tim, _ := time.Parse(layoutISO, row[0])
				_, err = stmt.Exec(queue.qu[q].TenantID, sliceLabel, tim, row[1])
				if err != nil {
					logging.Fatal(err)
				}
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
			queue.qu = append(queue.qu, model.DecodeToRequestReceived(data))
		} 
	}

	return nil
}


func createGRPCServer() {
	lis, err := net.Listen("tcp", ":9011")
	if err != nil {
		logging.Fatal(err)
	}
	logging.Println("listening on 9011")

	grpcServer := grpc.NewServer()
	pb.RegisterIngesterServiceServer(grpcServer, &server{})
	if err := grpcServer.Serve(lis); err != nil {
		logging.Fatal(err)
	}

	logging.Println("serving...")

}
