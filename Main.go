package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
)

func main() {
	caData, fErr := ioutil.ReadFile("./tls/rootCA.crt")
	if fErr != nil {
		log.Fatalln("Could not load rootCA: ", fErr)
		return
	}
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(caData)
	if !ok {
		log.Fatal("failed to parse root certificate")
		return
	}
	tlsConfig := &tls.Config{RootCAs: roots, ServerName: "localhost"}
	connection, err := net.Dial("tcp", "127.0.0.1:2750")

	if err != nil {
		log.Fatalln("CONNECTION ERROR: ", err)
	}
	conn := tls.Client(connection, tlsConfig)
	fmt.Println("Connection established")
	recvMsgChannel := make(chan *Message)
	disconnectChannel := make(chan *Client)
	client = &Client{
		connection: conn,
		sendChannel: make(chan *Message),
		recvChannel: recvMsgChannel,
		disconnectChannel: disconnectChannel,
	}
	go client.runSend()
	go client.runRead()
	go runNetEvents(recvMsgChannel, disconnectChannel)

	fmt.Println("InitChat")
	fmt.Println("---------------------")
	readAuthSelection()
}