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
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
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

var clear map[string]func() //create a map for storing clear funcs

func init() {
	clear = make(map[string]func()) //Initialize it
	clear["linux"] = func() {
		cmd := exec.Command("clear") //Linux example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	clear["windows"] = func() {
		cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func clearScreen() {
	value, ok := clear[runtime.GOOS] //runtime.GOOS -> linux, windows, darwin etc.
	if ok { //if we defined a clear func for that platform:
		value()  //we execute it
	} else { //unsupported platform
		panic("Your platform is unsupported! I can't clear terminal screen :(")
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
	clearScreen()
	fmt.Println("1) Sign Up")
	fmt.Println("2) Login")
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "1" {
		readSignUp(reader)
	} else if text == "2" {
		readLogin(reader)
	} else {
		fmt.Println("Invalid Input")
		readAuthSelection(reader)
	}
}

func readSignUp(reader *bufio.Reader) {
	clearScreen()
	fmt.Println("Enter ~cancel to leave")
	for {
		fmt.Println("Enter Username: ")
		username, _ := reader.ReadString('\n')
		username = strings.TrimSpace(username)
		if username == "~cancel" {
			readAuthSelection(reader)
		}
		fmt.Println("Enter Password: ")
		password, _ := reader.ReadString('\n')
		password = strings.TrimSpace(password)
		if password == "~cancel" {
			readAuthSelection(reader)
		}
		err := signUp(username, password)
		if err == nil {
			readHome(reader)
		}
		fmt.Println("SignUp Failed")
	}
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
	clearScreen()
	fmt.Println("Enter ~cancel to leave")
	for {
		fmt.Println("Enter Username: ")
		username, _ := reader.ReadString('\n')
		username = strings.TrimSpace(username)
		if username == "~cancel" {
			readAuthSelection(reader)
		}
		fmt.Println("Enter Password: ")
		password, _ := reader.ReadString('\n')
		password = strings.TrimSpace(password)
		if password == "~cancel" {
			readAuthSelection(reader)
		}
		err := login(username, password)
		if err == nil {
			readHome(reader)
		}
		fmt.Println("Login Failed")
		readLogin(reader)
	}
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

	select {
	case <-authChannel:
		return nil
	case <- errChannel:
		return errors.New("Login Failed")
	}
}

func readHome(reader *bufio.Reader) {
	clearScreen()
	fmt.Println("1) Create Chat Group\n" +
		"2) Open Previous Chat Group\n" +
		"3) View Invitations\n" +
		"4) Sign Out")
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "1" {
		readCreateGroup(reader)
	} else if text == "2" {
		readGroupList(reader)
	} else if text == "3" {
		readInvites(reader)
	} else if text == "4" {
		readAuthSelection(reader)
	} else {
		fmt.Println("Invalid Input")
		readAuthSelection(reader)
	}
}

func readCreateGroup(reader *bufio.Reader) {
	clearScreen()
	fmt.Println("Enter Group Name: ")
	groupName, _ := reader.ReadString('\n')
	groupName = strings.TrimSpace(groupName)
	groupData, err := createGroup(groupName)
	if err != nil {
		fmt.Println("Create Group Failed!")
		readCreateGroup(reader)
		return
	}
	readGroup(groupData, reader)
}

func createGroup(groupName string) ([]byte, error) {
	groupChannel := make(chan *Message)
	errChannel := make(chan *Message)
	msgHandlers["group"] = groupChannel
	msgHandlers["createGroupErr"] = errChannel
	defer func() {
		msgHandlers["group"] = nil
		msgHandlers["createGroupErr"] = nil
	}()

	createGroupMsg := Messages.CreateGroupReq{
		GroupName: groupName,
	}
	createGroupData, err := proto.Marshal(&createGroupMsg)
	if err != nil {
		log.Fatalln("Serialize Err: ", err)
		return nil, err
	}

	client.send("createGroup", createGroupData)

	select {
	case groupMsg := <-groupChannel:
		return groupMsg.body, nil
	case <- errChannel:
		return nil, errors.New("Create Group Failed")
	}
}

func listenForMessages() chan *Message {
	msgChannel := make(chan *Message)
	msgHandlers["message"] = msgChannel
	go func() {
		for {
			message, isMore := <-msgChannel
			if !isMore {
				return
			}
			textMsg := Messages.TextMessage{}
			parseErr := proto.Unmarshal(message.body, &textMsg)
			if parseErr != nil {
				log.Fatalln("PARSE ERR: ", parseErr)
				return
			}
			fmt.Println(textMsg.Username + ": " + textMsg.Message + "\n")
		}
	}()
	return msgChannel
}

func refreshGroup() ([]byte, error) {
	groupChannel := make(chan *Message)
	msgHandlers["group"] = groupChannel
	defer func() {
		msgHandlers["group"] = nil
	}()
	client.send("refreshGroup", nil)
	groupMsg := <- groupChannel
	return groupMsg.body, nil
}

func readGroup(groupData []byte, reader *bufio.Reader) {
	clearScreen()
	msgChannel := listenForMessages()
	fmt.Println("Commands:\n~invite\t#Invite a user\n" +
		"~leave\t#Leave the group\n" +
		"~fs\t#Send file")
	groupMsg := Messages.GroupResp{}
	proto.Unmarshal(groupData, &groupMsg)
	for _, textMsg := range groupMsg.Messages {
		fmt.Println(textMsg.Username + ": ", textMsg.Message + "\n")
	}

	for {
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if len(input) > 0 {
			if input[0] == '~' {
				if input == "~invite" {
					close(msgChannel)
					readInvite(reader)
				} else if input == "~leave" {
					leaveGroup()
					close(msgChannel)
					readHome(reader)
				} else if input == "~fs" {

				} else {
					fmt.Println("Invalid Command")
				}
			} else {
				sendTextMessage(input)
			}
		}
	}
}

func sendTextMessage(contents string) {
	textMsg := Messages.TextMessageReq{
		Message: contents,
	}
	textData, err:= proto.Marshal(&textMsg)
	if err != nil {
		log.Fatalln("SERIALIZE ERR: ", err)
		return
	}
	client.send("textMsg", textData)
}

func leaveGroup() {
	client.send("leaveGroup", nil)
}

func readInvite(reader *bufio.Reader) {
	clearScreen()
	fmt.Println("Search for user or enter command: \n" +
		"~cancel\t#Cancel Search\n" +
		"~invite <User #>\t#Invite user with number to group")
	var usernames []string
	for {
		fmt.Println("Enter search or command: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if len(input) > 0 {
			if input[0] == '~' {
				if input == "~cancel" {
					groupData, err := refreshGroup()
					if err == nil {
						readGroup(groupData, reader)
					}
					fmt.Println("RefreshGroup Error")
				} else if strings.Index(input, "~invite") == 0 {
					userNumStr := input[len("~invite"):]
					userNumStr = strings.TrimSpace(userNumStr)
					userI, convErr := strconv.Atoi(userNumStr)
					if convErr == nil {
						if userI >= 0 && userI < len(usernames) {
							var username = usernames[userI]
							inviteUser(username)
							groupData, err := refreshGroup()
							if err == nil {
								readGroup(groupData, reader)
							}
							fmt.Println("RefreshGroup Error")
						}
					} else {
						fmt.Println("Invalid User#")
					}
				} else {
					fmt.Println("Invalid Command")
				}
			} else {
				recvUsernames, err := searchUsers(input)
				if err != nil {
					fmt.Println("Search User ERROR")
				}
				usernames = recvUsernames
				if len(usernames) > 0 {
					fmt.Println("FOUND USERS: ")
					for i, username := range usernames {
						fmt.Println(i, ":", username)
					}
				} else {
					fmt.Println("No matching users found")
				}
			}
		}
	}
}

func searchUsers(namePrefix string) ([]string, error) {
	respChannel := make(chan *Message)
	errChannel := make(chan *Message)
	msgHandlers["userSearchResp"] = respChannel
	msgHandlers["userSearchErr"] = errChannel
	defer func() {
		msgHandlers["group"] = nil
		msgHandlers["createGroupErr"] = nil
	}()

	searchUserReq := Messages.UserSearchReq{
		UsernamePrefix: namePrefix,
	}
	reqData, serializeErr := proto.Marshal(&searchUserReq)
	if serializeErr != nil {
		log.Fatalln("SERIALIZE ERR: ", serializeErr)
		return nil, serializeErr
	}
	client.send("searchUsers", reqData)
	select {
	case respMsg := <-respChannel:
		resp := Messages.UserSearchResp{}
		proto.Unmarshal(respMsg.body, &resp)
		return resp.Usernames, nil
	case <- errChannel:
		return nil, errors.New("Search Groups Failed")
	}
}

func inviteUser(username string) {
	inviteUserReq := Messages.InviteReq{
		Username: username,
	}
	inviteUserData, serializeErr := proto.Marshal(&inviteUserReq)
	if serializeErr != nil {
		log.Fatalln("SERIALIZE ERR:", serializeErr)
	}
	client.send("invite", inviteUserData)
}

func readGroupList(reader *bufio.Reader) {
	clearScreen()

	groupNames, err := getGroupList()
	if err != nil {
		fmt.Println("Failed to get groups")
		time.Sleep(2 * time.Second)
		readHome(reader)
	}
	if len(groupNames) == 0 {
		fmt.Println("You Aren't In Any Groups...")
		time.Sleep(2 * time.Second)
		readHome(reader)
	}
	fmt.Println("Type ~cancel to leave\n" +
		"Enter group # to join: ")
	for i, groupName := range groupNames {
		fmt.Println(i, ": " + groupName)
	}
	for {
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if len(input) > 0 {
			if input[0] == '~' {
				if input == "~cancel" {
					readHome(reader)
				} else {
					fmt.Println("Invalid command")
				}
			} else {
				groupNum, convErr := strconv.Atoi(input)
				if convErr == nil {
					if groupNum >= 0 && groupNum < len(groupNames) {
						groupName := groupNames[groupNum]
						groupData, err := joinGroup(groupName)
						if err == nil {
							readGroup(groupData, reader)
						} else {
							fmt.Println("Join group failed")
						}
					} else {
						fmt.Println("Group# out of range")
					}
				} else {
					fmt.Println("Invalid integer")
				}
			}
		}
	}
}

func joinGroup(groupName string) ([]byte, error) {
	joinGroupMsg := Messages.JoinGroupReq{
		GroupName: groupName,
	}
	joinGroupData, serializeErr := proto.Marshal(&joinGroupMsg)
	if serializeErr != nil {
		log.Fatalln("SERIALIZE ERR: ", serializeErr)
		return nil, serializeErr
	}

	groupChannel := make(chan *Message)
	errChannel := make(chan *Message)
	msgHandlers["group"] = groupChannel
	msgHandlers["joinGroupErr"] = errChannel
	defer func() {
		msgHandlers["group"] = nil
		msgHandlers["joinGroupErr"] = nil
	}()
	client.send("joinGroup", joinGroupData)
	select {
	case groupMsg := <- groupChannel:
		return groupMsg.body, nil
	case <- errChannel:
		return nil, errors.New("Could not join group")
	}
}

func getGroupList() ([]string, error) {
	getGroupsChannel := make(chan *Message)
	errChannel := make(chan *Message)
	msgHandlers["getGroups"] = getGroupsChannel
	msgHandlers["getGroupsErr"] = errChannel
	defer func() {
		msgHandlers["getGroups"] = nil
		msgHandlers["getGroupsErr"] = nil
	}()
	client.send("getGroups", nil)
	select {
	case getGroupsMsg := <- getGroupsChannel:
		getGroupsResp := Messages.GroupsResp{}
		parseErr := proto.Unmarshal(getGroupsMsg.body, &getGroupsResp)
		if parseErr != nil {
			log.Fatalln("PARSE ERR: ", parseErr)
			return nil, parseErr
		}
		return getGroupsResp.GroupNames, nil
	case <- errChannel:
		return nil, errors.New("Could not list groups")
	}
}

func readInvites(reader *bufio.Reader) {
	clearScreen()
	invites, err := getInvites()
	if err != nil {
		fmt.Println("Could not get invites")
		readHome(reader)
	}
	fmt.Println("Type ~cancel to leave")
	fmt.Println("~accept <invite #>\t#Accept invite\n" +
		"~decline <invite #>\t#Decline invite\n" +
		"~refresh\t#Refresh invites")
	printInvites := func() {
		if len(invites) > 0 {
			for i, invite := range invites {
				fmt.Println(i, ": ", invite.GroupName + " from " + invite.FromUsername)
			}
		} else {
			fmt.Println("No Invites...")
		}
	}
	printInvites()
	for {
		fmt.Println("Enter command: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if len(input) > 0 {
			if input == "~cancel" {
				readHome(reader)
			} else if input == "~refresh" {
				recvInvites, err := getInvites()
				if err != nil {
					fmt.Println("Could not get invites")
					readHome(reader)
				}
				invites = recvInvites
				printInvites()
			} else if strings.Index(input, "~accept") == 0 {
				inviteNumStr := input[len("~accept"):]
				inviteNumStr = strings.TrimSpace(inviteNumStr)
				inviteI, convErr := strconv.Atoi(inviteNumStr)
				if convErr == nil {
					if inviteI >= 0 && inviteI < len(invites) {
						invite := invites[inviteI]
						recvInvites, err := acceptInvite(invite.InviteID)
						if err == nil {
							invites = recvInvites
							printInvites()
						} else {
							log.Println("Could not accept invite")
						}
					} else {
						log.Println("Invite # out of range")
					}
				} else {
					log.Println("Could not convert invite #")
				}
			} else if strings.Index(input, "~decline") == 0 {
				inviteNumStr := input[len("~decline"):]
				inviteNumStr = strings.TrimSpace(inviteNumStr)
				inviteI, convErr := strconv.Atoi(inviteNumStr)
				if convErr == nil {
					if inviteI >= 0 && inviteI < len(invites) {
						invite := invites[inviteI]
						recvInvites, err := declineInvite(invite.InviteID)
						if err == nil {
							invites = recvInvites
							printInvites()
						} else {
							log.Println("Could not decline invite")
						}
					} else {
						log.Println("Invite # out of range")
					}
				} else {
					log.Println("Could not convert invite #")
				}
			} else {
				fmt.Println("Invalid Command")
			}
		}
	}
}

func getInvites() ([]*Messages.InvitesResp_Invite, error) {
	getInvitesChannel := make(chan *Message)
	errChannel := make(chan *Message)
	msgHandlers["getInvites"] = getInvitesChannel
	msgHandlers["getInvitesErr"] = errChannel
	defer func() {
		msgHandlers["getInvites"] = nil
		msgHandlers["getInvitesErr"] = nil
	}()
	client.send("getInvites", nil)
	select {
	case getInvitesMsg := <- getInvitesChannel:
		getInvitesResp := Messages.InvitesResp{}
		parseErr := proto.Unmarshal(getInvitesMsg.body, &getInvitesResp)
		if parseErr != nil {
			log.Fatalln("PARSE ERR: ", parseErr)
			return nil, parseErr
		}
		return getInvitesResp.Invites, nil
	case <- errChannel:
		return nil, errors.New("Could not list groups")
	}
}

func acceptInvite(inviteID string) ([]*Messages.InvitesResp_Invite, error) {
	getInvitesChannel := make(chan *Message)
	errChannel := make(chan *Message)
	msgHandlers["getInvites"] = getInvitesChannel
	msgHandlers["acceptInviteErr"] = errChannel
	msgHandlers["getInvitesErr"] = errChannel
	defer func() {
		msgHandlers["getInvites"] = nil
		msgHandlers["acceptInviteErr"] = nil
		msgHandlers["getInvitesErr"] = nil
	}()
	acceptInviteReq := Messages.AcceptInviteReq{
		InviteID: inviteID,
	}
	acceptInviteData, serializeErr := proto.Marshal(&acceptInviteReq)
	if serializeErr != nil {
		log.Fatalln("SERIALIZE ERR: ", serializeErr)
		return nil, serializeErr
	}
	client.send("acceptInvite", acceptInviteData)
	select {
	case getInvitesMsg := <- getInvitesChannel:
		getInvitesResp := Messages.InvitesResp{}
		parseErr := proto.Unmarshal(getInvitesMsg.body, &getInvitesResp)
		if parseErr != nil {
			log.Fatalln("PARSE ERR: ", parseErr)
			return nil, parseErr
		}
		return getInvitesResp.Invites, nil
	case <- errChannel:
		return nil, errors.New("Could not accept invite")
	}
}

func declineInvite(inviteID string) ([]*Messages.InvitesResp_Invite, error) {
	getInvitesChannel := make(chan *Message)
	errChannel := make(chan *Message)
	msgHandlers["getInvites"] = getInvitesChannel
	msgHandlers["deleteInviteErr"] = errChannel
	msgHandlers["getInvitesErr"] = errChannel
	defer func() {
		msgHandlers["getInvites"] = nil
		msgHandlers["deleteInviteErr"] = nil
		msgHandlers["getInvitesErr"] = nil
	}()
	deleteInviteReq := Messages.DeleteInviteReq{
		InviteID: inviteID,
	}
	deleteInviteData, serializeErr := proto.Marshal(&deleteInviteReq)
	if serializeErr != nil {
		log.Fatalln("SERIALIZE ERR: ", serializeErr)
		return nil, serializeErr
	}
	client.send("deleteInvite", deleteInviteData)
	select {
	case getInvitesMsg := <- getInvitesChannel:
		getInvitesResp := Messages.InvitesResp{}
		parseErr := proto.Unmarshal(getInvitesMsg.body, &getInvitesResp)
		if parseErr != nil {
			log.Fatalln("PARSE ERR: ", parseErr)
			return nil, parseErr
		}
		return getInvitesResp.Invites, nil
	case <- errChannel:
		return nil, errors.New("Could not accept invite")
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