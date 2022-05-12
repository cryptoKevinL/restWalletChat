package entity

//Chat object for REST(CRUD)
type Inbox struct {
	//ID             int    `json:"id"`
	Address        string `json:"address"`
	Last_timestamp string `json:"last_timestamp"`
	Unread         string `json:"unread"`
	Message        string `json:"message"`
}
