package entity

type Comments struct {
	ID        int
	Fromaddr  string `json:"fromaddr"`
	Nftaddr   string `json:"nftaddr"`
	Nftid     int    `json:"nftid"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Name      string `json:"name"`
}
