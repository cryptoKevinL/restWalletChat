package controllers

import (
	"encoding/json"
	"io/ioutil"

	//"log"
	"net/http"
	"rest-go-demo/database"
	"rest-go-demo/entity"

	"github.com/gorilla/mux"
)

//GetAllInbox get all inboxes data
// func GetAllInbox(w http.ResponseWriter, r *http.Request) {
// 	var inbox []entity.Inbox
// 	database.Connector.Find(&inbox)
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(inbox)
// }

//GetInboxByID returns person with specific ID
func GetInboxByOwner(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["id"]

	var userInbox entity.Chatitem
	database.Connector.First(&userInbox, key)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userInbox)
}

//CreateInbox creates inbox
// func CreateInbox(w http.ResponseWriter, r *http.Request) {
// 	requestBody, _ := ioutil.ReadAll(r.Body)
// 	var inbox entity.Chatitem
// 	json.Unmarshal(requestBody, &inbox)

// 	database.Connector.Create(inbox)
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusCreated)
// 	json.NewEncoder(w).Encode(inbox)
// }

//UpdateInboxByOwner updates inbox with respective owner address
// func UpdateInboxByOwner(w http.ResponseWriter, r *http.Request) {
// 	requestBody, _ := ioutil.ReadAll(r.Body)
// 	var inbox entity.Chatitem
// 	json.Unmarshal(requestBody, &inbox)
// 	database.Connector.Save(&inbox)

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(inbox)
// }

//DeletePersonByID delete's person with specific ID
// func DeleteInboxByOwner(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	key := vars["address"]

// 	var inbox entity.Inbox
// 	//id, _ := strconv.ParseString(key, 10, 64)
// 	database.Connector.Where("address = ?", key).Delete(&inbox)
// 	w.WriteHeader(http.StatusNoContent)
// }

//*********chat info*********************
//GetAllChatitems get all chat data
func GetAllChatitems(w http.ResponseWriter, r *http.Request) {
	//log.Println("get all chats")
	var chat []entity.Chatitem
	database.Connector.Find(&chat)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(chat)
}

//GetChatFromAddressToOwner returns all chat items from user to owner
func GetChatFromAddress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["address"]

	var chat []entity.Chatitem
	database.Connector.Where("fromaddr = ?", key).Or("toaddr = ?", key).Find(&chat)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chat)
}

//CreateChatitem creates Chatitem
func CreateChatitem(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var chat entity.Chatitem
	json.Unmarshal(requestBody, &chat)

	database.Connector.Create(chat)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(chat)
}

//UpdateInboxByOwner updates person with respective owner address
func UpdateChatitemByOwner(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var chat entity.Chatitem
	//database.Connector.Where("fromaddr = ?", owner).Where("toaddr = ?", to).Find(&chat)
	json.Unmarshal(requestBody, &chat)
	database.Connector.Save(&chat)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(chat)
}

func DeleteAllChatitemsToAddressByOwner(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	to := vars["toAddr"]
	owner := vars["fromAddr"]

	var chat entity.Chatitem
	//id, _ := strconv.ParseString(key, 10, 64)
	database.Connector.Where("toAddr = ?", to).Where("fromAddr = ?", owner).Delete(&chat)
	w.WriteHeader(http.StatusNoContent)
}
