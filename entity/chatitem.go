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

//changing case causes _ in Golang table name calls....thats why its all lower case after first char
type Groupchatitem struct {
	Fromaddr  string    `json:"fromaddr"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Nftaddr   string    `json:"nftaddr"`
}

//have to make a new version of the table with type, for walletchat living room welcome messsages
//can convert over to this fully when released in store with UI changes (this helps current store verisons keep working)
type V2groupchatitem struct {
	Fromaddr  string    `json:"fromaddr"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Nftaddr   string    `json:"nftaddr"`
	Type      string    `json:"type"`
}

//secondary table to help only load new messages for each user (not reload whole chat history)
type Groupchatreadtime struct {
	Fromaddr      string    `json:"fromaddr"`
	Lasttimestamp time.Time `json:"lasttimestamp"`
	Nftaddr       string    `json:"nftaddr"`
}

//potentially use this to keep track of user logins for DAU metrics
type Logintime struct {
	Address   string    `json:"address"`
	Timestamp time.Time `json:"timestamp"`
}

type Addrnameitem struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type Bookmarkitem struct {
	Walletaddr string `json:"walletaddr"`
	Nftaddr    string `json:"nftaddr"`
}

type BookmarkReturnItem struct {
	Walletaddr    string    `json:"walletaddr"`
	Nftaddr       string    `json:"nftaddr"`
	Lastmsg       string    `json:"lastmsg"`
	Lasttimestamp time.Time `json:"lasttimestamp"`
	Unreadcnt     int       `json:"unreadcnt"`
}

type Nftsidebar struct {
	Fromaddr string `json:"fromaddr"`
	Unread   int    `json:"unread"`
	Nftaddr  string `json:"nftaddr"`
	Nftid    int    `json:"nftid"`
}

type Chatiteminbox struct {
	Fromaddr   string `json:"fromaddr"`
	Toaddr     string `json:"toaddr"`
	Timestamp  string `json:"timestamp"`
	Msgread    bool   `json:"read"`
	Message    string `json:"message"`
	Nftaddr    string `json:"nftaddr"`
	Nftid      int    `json:"nftid"`
	Unreadcnt  int    `json:"unread"`
	Type       string `json:"type"`
	Sendername string `json:"sender_name"`
}
