package main

import (
	"./Messages"
	"bufio"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"log"
	"net"
	"os"
	"strings"
)

type MsgHandler func(message *Message)
var msgHandlers = map[string](chan *Message){}
var client *Client

func handleMessage(message *Message) {
	handler, containsHandler := msgHandlers[message.typeID]
	if containsHandler {
		handler <- message
	} else {
		log.Println("No type handler for id: ", message.typeID)
	}
}

func runNetEvents(recvMsgChannel chan *Message, disconnectChannel chan *Client) {
	for {
		select {
		case msg, more := <- recvMsgChannel:
			if !more {
				return
			}
			handleMessage(msg)
		case _, more := <- disconnectChannel:
			if !more {
				return
			}
			log.Println("DISCONNECTED")
		}
	}
}

func readAuthSelection(_ *bufio.Reader) {
	fmt.Println("1) Sign Up")
	fmt.Println("2) Login")
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	log.Println("TEXT: ", text)
	if text == "1" {
		readSignUp(reader)
	} else if text == "2" {
		readLogin(reader)
	} else {
		log.Println("Invalid Input")
		readAuthSelection(reader)
	}
}

func readSignUp(reader *bufio.Reader) {
	fmt.Println("Enter Username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)
	fmt.Println("Enter Password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)
	err := signUp(username, password)
	if err != nil {
		log.Println("SignUp Failed")
		readSignUp(reader)
		return
	}
	log.Println("SignUp Successful")
	readHome(reader)
}

func signUp(username string, password string) error {
	authChannel := make(chan *Message)
	errChannel := make(chan *Message)
	msgHandlers["auth"] = authChannel
	msgHandlers["authErr"] = errChannel
	defer func() {
		msgHandlers["auth"] = nil
		msgHandlers["authErr"] = nil
	}()

	signUpMsg := Messages.SignUpReq{
		Username: username,
		Password: password,
	}
	signUpData, err := proto.Marshal(&signUpMsg)
	if err != nil {
		log.Fatalln("Serialize Err: ", err)
		return err
	}

	client.send("signUp", signUpData)

	select {
	case <-authChannel:
		return nil
	case <- errChannel:
		return errors.New("SignUp Failed")
	}
}


func readLogin(reader *bufio.Reader) {
	fmt.Println("Enter Username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)
	fmt.Println("Enter Password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)
	err := login(username, password)
	if err != nil {
		log.Println("Login Failed")
		readLogin(reader)
		return
	}
	log.Println("Login Successful")
	readHome(reader)
}

func login(username string, password string) error {
	authChannel := make(chan *Message)
	errChannel := make(chan *Message)
	msgHandlers["auth"] = authChannel
	msgHandlers["authErr"] = errChannel
	defer func() {
		msgHandlers["auth"] = nil
		msgHandlers["authErr"] = nil
	}()

	loginMsg := Messages.LoginReq{
		Username: username,
		Password: password,
	}
	loginData, err := proto.Marshal(&loginMsg)
	if err != nil {
		log.Fatalln("Serialize Err: ", err)
		return err
	}

	client.send("login", loginData)
	log.Println("LOGIN SENT")

	select {
	case <-authChannel:
		return nil
	case <- errChannel:
		return errors.New("Login Failed")
	}
}

func readHome(reader *bufio.Reader) {
	log.Println("1) Create Chat Group\n" +
		"2) Open Previous Chat Group\n" +
		"3) View Invitations")
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	log.Println("TEXT: ", text)
	if text == "1" {

	} else if text == "2" {

	} else {
		log.Println("Invalid Input")
		readAuthSelection(reader)
	}
}

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:2750")
	if err != nil {
		log.Fatalln("CONNECTION ERROR: ", err)
	}
	recvMsgChannel := make(chan *Message)
	disconnectChannel := make(chan *Client)
	client = &Client{
		connection: &conn,
		sendChannel: make(chan *Message),
		recvChannel: recvMsgChannel,
		disconnectChannel: disconnectChannel,
	}
	go client.runSend()
	go client.runRead()
	go runNetEvents(recvMsgChannel, disconnectChannel)
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("InitChat")
	fmt.Println("---------------------")
	readAuthSelection(reader)
}