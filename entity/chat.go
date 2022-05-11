package entity

//INBOX
// {
// 	"address": "0xF0030495802f8f90Ace6d869aBd653f2062fD1De", // who are we chatting with?
// 	"last_timestamp": "2022-05-11T12:04:00.088Z", // date of last message, unread or read
// 	"unread": 0, // how many unread messages
// 	"message": "Aenean laoreet pretium dignissim. Vestibulum nunc magna, tincidunt euismod purus ac, porta eleifend ligula. Sed a rutrum sapien, vitae placerat ex. Nunc at ultricies dui." // last read or unread message
// },

//Chat object for REST(CRUD)
type Inbox struct {
	Address        string `json:"address"`
	Last_timestamp string `json:"last_timestamp"`
	Unread         int    `json:"unread"`
	Message        string `json:"message"`
}

// {
// 	"fromAddr": "0xF0030495802f8f90Ace6d869aBd653f2062fD1De",
// 	"toAddr": "0xDAB141eFC7Df3f3d1a97C06568140b2859F9BaC0",
// 	"timestamp": "2022-05-11T12:04:00.088Z", // date of last message, unread or read
// 	"read": true, // has the recipient/sender read the message?
// 	"message": "Aenean laoreet pretium dignissim. Vestibulum nunc magna, tincidunt euismod purus ac, porta eleifend ligula. Sed a rutrum sapien, vitae placerat ex. Nunc at ultricies dui." // last read or unread message
// },

type ChatItem struct {
	FromAddr  string `json:"fromAddr"`
	ToAddr    string `json:"toAddr"`
	Timestamp string `json:"timestamp"`
	Read      string `json:"read"`
	Message   string `json:"message"`
}
