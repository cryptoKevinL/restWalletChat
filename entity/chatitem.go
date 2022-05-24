package entity

type Chatitem struct {
	Fromaddr  string `json:"fromaddr"`
	Toaddr    string `json:"toaddr"`
	Timestamp string `json:"timestamp"`
	Msgread   bool   `json:"read"`
	Message   string `json:"message"`
	Nftaddr   string `json:"nftaddr"`
	Nftid     int    `json:"nftid"`
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
