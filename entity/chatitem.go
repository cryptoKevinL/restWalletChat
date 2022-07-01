package entity

import "time"

type Chatitem struct {
	Fromaddr  string `json:"fromaddr"`
	Toaddr    string `json:"toaddr"`
	Timestamp string `json:"timestamp"`
	Msgread   bool   `json:"read"`
	Message   string `json:"message"`
	Nftaddr   string `json:"nftaddr"`
	Nftid     int    `json:"nftid"`
}

//changing case causes _ in Golang table name calls....confused
type Groupchatitem struct {
	Fromaddr  string    `json:"fromaddr"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Nftaddr   string    `json:"nftaddr"`
}

//secondary table to help only load new messages for each user (not reload whole chat history)
type Groupchatreadtime struct {
	Fromaddr      string    `json:"fromaddr"`
	Lasttimestamp time.Time `json:"lasttimestamp"`
	Nftaddr       string    `json:"nftaddr"`
}

type Bookmarkitem struct {
	Walletaddr string `json:"walletaddr"`
	Nftaddr    string `json:"nftaddr"`
}

type Nftsidebar struct {
	Fromaddr string `json:"fromaddr"`
	Unread   int    `json:"unread"`
	Nftaddr  string `json:"nftaddr"`
	Nftid    int    `json:"nftid"`
}

type Chatiteminbox struct {
	Fromaddr  string `json:"fromaddr"`
	Toaddr    string `json:"toaddr"`
	Timestamp string `json:"timestamp"`
	Msgread   bool   `json:"read"`
	Message   string `json:"message"`
	Nftaddr   string `json:"nftaddr"`
	Nftid     int    `json:"nftid"`
	Unreadcnt int    `json:"unread"`
}

// type ChatitemRsp struct {
// 	ID        int    `json:"id"`
// 	Fromaddr  string `json:"fromaddr"`a
// 	Toaddr    string `json:"toaddr"`
// 	Timestamp string `json:"timestamp"`
// 	Read    string `json:"read"`
// 	Message   string `json:"message"`
// }
