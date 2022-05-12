package entity

type Chatitem struct {
	//ID        int    `json:"id"`
	Fromaddr  string `json:"fromaddr"`
	Toaddr    string `json:"toaddr"`
	Timestamp string `json:"timestamp"`
	Unread    string `json:"unread"`
	Message   string `json:"message"`
}
