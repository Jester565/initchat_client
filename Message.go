package main

type Message struct {
	typeID string
	body []byte
	client *Client
}