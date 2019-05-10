package main

import (
	"./Messages"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

var msgHandlers = map[string]chan *Message{}
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

func createGroup(groupName string) (*Messages.GroupResp, error) {
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
		group := Messages.GroupResp{}
		proto.Unmarshal(groupMsg.body, &group)
		return &group, nil
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
			t := time.Unix(int64(textMsg.Time), 0)
			fmt.Println("[" + t.Format("3:04PM") + "] " + textMsg.Username + " >> ", textMsg.Message + "\n")
		}
	}()
	return msgChannel
}

func refreshGroup() (*Messages.GroupResp, error) {
	groupChannel := make(chan *Message)
	msgHandlers["group"] = groupChannel
	defer func() {
		msgHandlers["group"] = nil
	}()
	client.send("refreshGroup", nil)
	groupMsg := <- groupChannel
	group := Messages.GroupResp{}
	proto.Unmarshal(groupMsg.body, &group)
	return &group, nil
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

func uploadFile(filePath string) {
	fileData, err := ioutil.ReadFile(filePath)
	if err == nil {
		fileNameStartI := strings.LastIndex(filePath, "/")
		if fileNameStartI < 0 {
			fileNameStartI = 0
		} else {
			fileNameStartI++
		}
		fileName := filePath[fileNameStartI:]
		uploadMsg := Messages.FileMessageReq{
			Name: fileName,
			Contents: fileData,
		}
		reqData, _ := proto.Marshal(&uploadMsg)
		client.send("upload", reqData)
	} else {
		fmt.Println("Could not load file")
	}
}

func downloadFile(fileID string) error {
	downloadReqMsg := Messages.DownloadReq{
		FileID: fileID,
	}
	downloadReqData, _ := proto.Marshal(&downloadReqMsg)

	downloadChannel := make(chan *Message)
	errChannel := make(chan *Message)
	msgHandlers["downloadResp"] = downloadChannel
	msgHandlers["downloadErr"] = errChannel
	defer func() {
		msgHandlers["downloadResp"] = nil
		msgHandlers["downloadErr"] = nil
	}()

	client.send("download", downloadReqData)

	select {
	case downloadMsg := <-downloadChannel:
		downloadResp := Messages.DownloadResp{}
		proto.Unmarshal(downloadMsg.body, &downloadResp)
		os.Mkdir("./downloads", 0644)
		filePath := "./downloads/" + downloadResp.FileID
		ioutil.WriteFile(filePath, downloadResp.Contents, 0644)
		return nil
	case <-errChannel:
		return errors.New("Failed to download file")
	}
}

func leaveGroup() {
	client.send("leaveGroup", nil)
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

func joinGroup(groupName string) (*Messages.GroupResp, error) {
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
		group := Messages.GroupResp{}
		proto.Unmarshal(groupMsg.body, &group)
		return &group, nil
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
		return nil, errors.New("could not list groups")
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