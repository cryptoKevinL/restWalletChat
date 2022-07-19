package entity

import "time"

//rename the type in GET /inbox to context_type: [nft, community, dm] and
//retain variable name type in message objects in communities to be [welcome, message] instead of [communitymsg, communitywelcome]
// string mapping
const ( //context_type mapping just for bookkeeping(golang sucks for enums as well...)
	Nft       string = "nft"
	Community string = "community"
	DM        string = "dm"
	All       string = "all"
)
const ( //type mapping just for bookkeeping(golang sucks for enums as well...)
	Welcome string = "welcome"
	Message string = "message"
)

type Unreadcountitem struct {
	//ID         int
	Walletaddr string `json:"walletaddr"`
	Nft        bool   `json:"nft"`
	Dm         bool   `json:"dm"`
	Community  bool   `json:"community"`
}

type Chatitem struct {
	ID            int
	Fromaddr      string    `json:"fromaddr"`
	Toaddr        string    `json:"toaddr"`
	Timestamp     string    `json:"timestamp"`
	Timestamp_dtm time.Time `json:"timestamp_dtm"`
	Msgread       bool      `json:"read"`
	Message       string    `json:"message"`
	Nftaddr       string    `json:"nftaddr"`
	Nftid         int       `json:"nftid"`
	Name          string    `json:"sender_name"`
}

//for olivers view function
type V_chatitem struct {
	ID            int
	Fromaddr      string    `json:"fromaddr"`
	Toaddr        string    `json:"toaddr"`
	Timestamp     string    `json:"timestamp"`
	Timestamp_dtm time.Time `json:"timestamp_dtm"`
	Msgread       bool      `json:"read"`
	Message       string    `json:"message"`
	Nftaddr       string    `json:"nftaddr"`
	Nftid         int       `json:"nftid"`
	Name          string    `json:"sender_name"`
}

//changing case causes _ in Golang table name calls....thats why its all lower case after first char
type Groupchatitem struct {
	ID            int
	Fromaddr      string    `json:"fromaddr"`
	Timestamp     string    `json:"timestamp"`
	Timestamp_dtm time.Time `json:"timestamp_dtm"`
	Message       string    `json:"message"`
	Nftaddr       string    `json:"nftaddr"`
	Type          string    `json:"type"`
	Contexttype   string    `json:"context_type"`
	Name          string    `json:"sender_name"`
}

//secondary table to help only load new messages for each user (not reload whole chat history)
type Groupchatreadtime struct {
	ID                int
	Fromaddr          string    `json:"fromaddr"`
	Readtimestamp_dtm time.Time `json:"readtimestamp_dtm"`
	Nftaddr           string    `json:"nftaddr"`
}

//potentially use this to keep track of user logins for DAU metrics
type Logintime struct {
	ID        int
	Address   string    `json:"address"`
	Timestamp time.Time `json:"timestamp"`
}

type Addrnameitem struct {
	ID      int
	Address string `json:"address"`
	Name    string `json:"name"`
}

type Imageitem struct {
	ID         int
	Base64data string `json:"base64data"`
	Name       string `json:"name"`
}

type Bookmarkitem struct {
	ID         int
	Walletaddr string `json:"walletaddr"`
	Nftaddr    string `json:"nftaddr"`
}

type BookmarkReturnItem struct {
	ID                int
	Walletaddr        string    `json:"walletaddr"`
	Nftaddr           string    `json:"nftaddr"`
	Lastmsg           string    `json:"lastmsg"`
	Lasttimestamp     string    `json:"lasttimestamp"`
	Lasttimestamp_dtm time.Time `json:"lasttimestamp_dtm"`
	Unreadcnt         int       `json:"unreadcnt"`
}

type Nftsidebar struct {
	ID       int
	Fromaddr string `json:"fromaddr"`
	Unread   int    `json:"unread"`
	Nftaddr  string `json:"nftaddr"`
	Nftid    int    `json:"nftid"`
}

//this is a return type only
type Chatiteminbox struct {
	ID            int
	Fromaddr      string    `json:"fromaddr"`
	Toaddr        string    `json:"toaddr"`
	Timestamp     string    `json:"timestamp"`
	Timestamp_dtm time.Time `json:"timestamp_dtm"`
	Msgread       bool      `json:"read"`
	Message       string    `json:"message"`
	Nftaddr       string    `json:"nftaddr"`
	Nftid         int       `json:"nftid"`
	Unreadcnt     int       `json:"unread"`
	Type          string    `json:"type"`
	Contexttype   string    `json:"context_type"`
	Sendername    string    `json:"sender_name"`
	Name          string    `json:"name"`
	LogoData      string    `json:"logo"`
}
