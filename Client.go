package main

import (
	"./Messages"
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	"io"
	"log"
)

var PreHeaderLength = 2

type Message struct {
	typeID string
	body []byte
	client *Client
}

type Client struct {
	connection *tls.Conn
	sendChannel chan *Message
	recvChannel chan *Message
	disconnectChannel chan *Client
}

func (client *Client) send(typeID string, body []byte) {
	msg := Message {
		typeID: typeID,
		body: body,
	}
	client.sendChannel <- &msg
}

func (client *Client) runSend() {
	fmt.Println("Running send")
	for {
		msg := <-client.sendChannel
		bodySize := int32(0)
		if msg.body != nil {
			bodySize = int32(len(msg.body))
		}
		header := &Messages.Header{Id: msg.typeID, Length: bodySize}
		headerData, err := proto.Marshal(header)
		if err != nil {
			log.Fatal("Header Serialization Failed: ", err)
			return
		}
		headerSize := uint16(len(headerData))
		preHeaderData := make([]byte, PreHeaderLength)
		binary.BigEndian.PutUint16(preHeaderData, headerSize)
		data := append(preHeaderData, headerData...)
		if msg.body != nil {
			data = append(data, msg.body...)
		}
		_, writeErr := client.connection.Write(data)
		//nBytes, writeErr := writer.Write(data)
		if writeErr != nil {
			log.Println("WriteErr: ", writeErr)
			return
		}
	}
}

func (client *Client) onDisconnect() {
	client.disconnectChannel <- client
}

func (client *Client) runRead() {
	reader := bufio.NewReader((*client).connection)
	defer client.onDisconnect()
	for {
		preHeaderData := make([]byte, PreHeaderLength)
		_, preHeaderErr := io.ReadFull(reader, preHeaderData)
		if preHeaderErr != nil {
			log.Println("Client Disconnected: ", preHeaderErr)
			return
		}
		headerSize := binary.BigEndian.Uint16(preHeaderData)

		headerData := make([]byte, headerSize)
		_, headerErr := io.ReadFull(reader, headerData)
		if headerErr != nil {
			log.Print("Client Disconnected: ", headerErr)
			return
		}
		header := &Messages.Header{}
		if parseErr := proto.Unmarshal(headerData, header); parseErr != nil {
			log.Print("Header Parse Error", parseErr)
			return
		}
		typeID := header.GetId()
		bodySize := header.GetLength()
		bodyData := make([]byte, bodySize)
		if bodySize > 0 {
			_, bodyErr := io.ReadFull(reader, bodyData)
			if bodyErr != nil {
				log.Print("Client Disconnected", bodyErr)
				return
			}
		}

		message := Message{typeID, bodyData, client}
		client.recvChannel <- &message
	}
}