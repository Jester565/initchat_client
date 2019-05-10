/*
	Handles the display flow
 */

package main

import (
	"./Messages"
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var Reader = bufio.NewReader(os.Stdin)

func readString(prompts ...interface{}) string {
	for _, elm := range prompts {
		fmt.Println(elm)
	}
	text, _ := Reader.ReadString('\n')
	return strings.TrimSpace(text)
}

//Clears entire terminal
func clearScreen() {
	if runtime.GOOS == "linux" {
		cmd := exec.Command("clear") //Command to clear linux terminal
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls") //Clears windows command
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		println("Could not clear screen")
	}
}

func readAuthSelection() {
	clearScreen()
	for {
		selection := readString("1) Sign Up", "2) Login", "3) Exit")
		switch selection {
		case "1":
			readSignUp()
		case "2":
			readLogin()
		case "3":
			return
		default:
			fmt.Println("Invalid Input")
		}
	}
}

func readSignUp() {
	clearScreen()
	fmt.Println("Enter ~cancel to leave")
	for {
		username := readString("Enter Username: ")
		if username == "~cancel" {
			clearScreen()
			return
		}
		password := readString("Enter Password: ")
		if password == "~cancel" {
			clearScreen()
			return
		}
		err := signUp(username, password)
		if err == nil {
			readHome()
			return
		}
		fmt.Println("SignUp Failed")
	}
}

func readLogin() {
	clearScreen()
	fmt.Println("Enter ~cancel to leave")
	for {
		username := readString("Enter Username: ")
		if username == "~cancel" {
			clearScreen()
			return
		}
		password := readString("Enter Password: ")
		if password == "~cancel" {
			clearScreen()
			return
		}
		err := login(username, password)
		//If login successful, show home display
		if err == nil {
			readHome()
			return
		}
		fmt.Println("Login Failed")
	}
}

func readHome() {
	clearScreen()
	for {
		selection := readString(
			"1) Create Chat Group",
			"2) Open Previous Chat Group",
			"3) View Invitations",
			"4) Sign Out")

		switch selection {
		case "1":
			readCreateGroup()
		case "2":
			readGroupList()
		case "3":
			readInvites()
		case "4":
			clearScreen()
			return
		default:
			fmt.Println("Invalid Input")
		}
	}
}

func readCreateGroup() {
	clearScreen()
	fmt.Println("Enter ~cancel to go back")
	for {
		groupName := readString("Enter Group Name: ")
		if groupName == "~cancel" {

		}
		group, err := createGroup(groupName)
		if err == nil {
			readGroup(*group)
			return
		}
		fmt.Println("Create Group Failed!")
	}
}

func readGroup(groupMsg Messages.GroupResp) {
	clearScreen()
	msgChannel := listenForMessages()
	fmt.Println("Commands:\n~invite\t#Invite a user\n" +
		"~leave\t#Leave the group\n" +
		"~upload {path}\t#Send file\n" +
		"~download {fileID}\t#Download file")
	for _, textMsg := range groupMsg.Messages {
		t := time.Unix(int64(textMsg.Time), 0)
		fmt.Println("[" + t.Format("3:04PM") + "] " + textMsg.Username + " >> ", textMsg.Message + "\n")
	}

	for {
		input := readString()
		if len(input) > 0 {
			if input[0] == '~' {
				if input == "~invite" {
					close(msgChannel)
					readInvite()
				} else if input == "~leave" {
					leaveGroup()
					close(msgChannel)
					clearScreen()
					return
				} else if strings.Index(input, "~upload") == 0 {
					pathStr := input[len("~upload"):]
					pathStr = strings.TrimSpace(pathStr)
					uploadFile(pathStr)
				} else if strings.Index(input, "~download") == 0 {
					fileID := input[len("~download"):]
					fileID = strings.TrimSpace(fileID)
					downloadErr := downloadFile(fileID)
					if downloadErr != nil {
						fmt.Println(downloadErr)
					} else {
						fmt.Println("Download Successful!")
					}
				} else {
					fmt.Println("Invalid Command")
				}
			} else {
				sendTextMessage(input)
			}
		}
	}
}

func readInvite() {
	clearScreen()
	fmt.Println("Search for user or enter command: \n" +
		"~cancel\t#Cancel Search\n" +
		"~invite <User #>\t#Invite user with number to group")
	var usernames []string
	for {
		input := readString("Enter search or command: ")
		if len(input) > 0 {
			if input[0] == '~' {
				if input == "~cancel" {
					group, err := refreshGroup()
					if err == nil {
						readGroup(*group)
						return
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
							group, err := refreshGroup()
							if err == nil {
								readGroup(*group)
								return
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

func readGroupList() {
	clearScreen()

	groupNames, err := getGroupList()
	if err != nil {
		fmt.Println("Failed to get groups")
		return
	}
	if len(groupNames) == 0 {
		fmt.Println("You Aren't In Any Groups...")
		return
	}
	fmt.Println("Type ~cancel to leave")
	for i, groupName := range groupNames {
		fmt.Println(i, ": " + groupName)
	}
	for {
		input := readString("Enter Group # to Join: ")
		if len(input) > 0 {
			if input[0] == '~' {
				if input == "~cancel" {
					clearScreen()
					return
				} else {
					fmt.Println("Invalid command")
				}
			} else {
				groupNum, convErr := strconv.Atoi(input)
				if convErr == nil {
					if groupNum >= 0 && groupNum < len(groupNames) {
						groupName := groupNames[groupNum]
						group, err := joinGroup(groupName)
						if err == nil {
							readGroup(*group)
							return
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

func printInvites(invites []*Messages.InvitesResp_Invite) {
	if len(invites) > 0 {
		for i, invite := range invites {
			fmt.Println(i, ": ", invite.GroupName + " from " + invite.FromUsername)
		}
	} else {
		fmt.Println("No Invites...")
	}
}

func readInvites() {
	clearScreen()
	invites, err := getInvites()
	if err != nil {
		fmt.Println("Could not get invites")
		return
	}
	fmt.Println("Type ~cancel to leave\n" +
		"~accept <invite #>\t#Accept invite\n" +
		"~decline <invite #>\t#Decline invite\n" +
		"~refresh\t#Refresh invites")
	printInvites(invites)
	for {
		input := readString("Enter command: ")
		if len(input) > 0 {
			if input == "~cancel" {
				clearScreen()
				return
			} else if input == "~refresh" {
				recvInvites, err := getInvites()
				if err != nil {
					fmt.Println("Could not get invites")
					return
				}
				invites = recvInvites
				printInvites(invites)
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
							printInvites(invites)
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
							printInvites(invites)
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
