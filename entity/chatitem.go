package entity

type Chatitem struct {
	Fromaddr  string `json:"fromaddr"`
	Toaddr    string `json:"toaddr"`
	Timestamp string `json:"timestamp"`
	Msgread   bool   `json:"read"`
	Message   string `json:"message"`
}

// type ChatitemRsp struct {
// 	ID        int    `json:"id"`
// 	Fromaddr  string `json:"fromaddr"`
// 	Toaddr    string `json:"toaddr"`
// 	Timestamp string `json:"timestamp"`
// 	Read    string `json:"read"`
// 	Message   string `json:"message"`
// }
