package entity

type Chatitem struct {
	Fromaddr  string `json:"fromaddr"`
	Toaddr    string `json:"toaddr"`
	Timestamp string `json:"timestamp"`
	Unread    string `json:"unread"`
	Message   string `json:"message"`
}

// type ChatitemRsp struct {
// 	ID        int    `json:"id"`
// 	Fromaddr  string `json:"fromaddr"`
// 	Toaddr    string `json:"toaddr"`
// 	Timestamp string `json:"timestamp"`
// 	Unread    string `json:"unread"`
// 	Message   string `json:"message"`
// }
