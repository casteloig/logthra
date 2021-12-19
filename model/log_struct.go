package model

import (
	"bytes"
	"encoding/gob"
	logging "log"
)

type RequestReceived struct {
	TenantID string
	Streams []StreamStruct `json:"streams"` 
}

type StreamStruct struct {
	Stream map[string]string `json:"stream"`
	Values [][2]string `json:"values"`
}


func DecodeToRequestReceived(s []byte) RequestReceived {

	r := RequestReceived{}
	dec := gob.NewDecoder(bytes.NewReader(s))
	err := dec.Decode(&r)
	if err != nil {
		logging.Fatal(err)
	}

	return r
}

func EncodeToBytes(p interface{}) []byte  {

	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(p)
	if err != nil {
		logging.Fatal(err)
	}
	
	return buf.Bytes()
}