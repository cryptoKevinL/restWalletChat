package entity

//Settings object for REST(CRUD)
type Settings struct {
	ID         int    `json:"id"`
	Walletaddr string `json:"walletaddr"`
	//Publickey  string `json:"publickey"` //need this for encryption, don't want to get it over and over
	Email string `json:"email"`
	//Allow_read_rx string `json:"allow_read_rx"`
}
