package controllers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"rest-go-demo/database"
	"rest-go-demo/entity"

	"github.com/gorilla/mux"
)

//GetAllInbox get all inboxes data
func GetAllInbox(w http.ResponseWriter, r *http.Request) {
	var inbox []entity.Inbox
	database.Connector.Find(&inbox)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(inbox)
}

//GetInboxByID returns person with specific ID
func GetInboxByOwner(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["address"]

	var userInbox entity.Inbox
	database.Connector.First(&userInbox, key)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userInbox)
}

//CreateInbox creates inbox
func CreateInbox(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var inbox entity.Inbox
	json.Unmarshal(requestBody, &inbox)

	database.Connector.Create(inbox)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(inbox)
}

//UpdateInboxByOwner updates inbox with respective owner address
func UpdateInboxByOwner(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var inbox entity.Inbox
	json.Unmarshal(requestBody, &inbox)
	database.Connector.Save(&inbox)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(inbox)
}

//DeletePersonByID delete's person with specific ID
func DeleteInboxByOwner(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["address"]

	var inbox entity.Inbox
	//id, _ := strconv.ParseString(key, 10, 64)
	database.Connector.Where("address = ?", key).Delete(&inbox)
	w.WriteHeader(http.StatusNoContent)
}

//*********chat info*********************
//GetAllChatItems get all chat data
func GetAllChatItems(w http.ResponseWriter, r *http.Request) {
	var chatItem []entity.ChatItem
	database.Connector.Find(&chatItem)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(chatItem)
}

//GetChatFromAddressToOwner returns all chat items from user to owner
func GetChatFromAddressToOwner(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	to := vars["toaddr"]
	owner := vars["fromaddr"]

	var chatItem []entity.ChatItem
	database.Connector.Where("toAddr = ?", to).Where("fromAddr = ?", owner).Find(&chatItem)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatItem)
}

//CreateChatItem creates ChatItem
func CreateChatItem(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var chatItem entity.ChatItem
	json.Unmarshal(requestBody, &chatItem)

	database.Connector.Create(chatItem)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(chatItem)
}

//UpdateInboxByOwner updates person with respective owner address
func UpdateChatItemByOwner(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var chatItem entity.ChatItem
	json.Unmarshal(requestBody, &chatItem)
	database.Connector.Save(&chatItem)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(chatItem)
}

func DeleteAllChatItemsToAddressByOwner(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	to := vars["toAddr"]
	owner := vars["fromAddr"]

	var chatItem entity.ChatItem
	//id, _ := strconv.ParseString(key, 10, 64)
	database.Connector.Where("toAddr = ?", to).Where("fromAddr = ?", owner).Delete(&chatItem)
	w.WriteHeader(http.StatusNoContent)
}
