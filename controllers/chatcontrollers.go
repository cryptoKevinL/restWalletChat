package controllers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"rest-go-demo/database"
	"rest-go-demo/entity"
	"time"

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

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

//GetInboxByID returns the latest message for each unique conversation
//TODO: properly design the relational DB structs to optimize this search/retrieve
func GetInboxByOwner(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["address"] //owner of the inbox

	//fmt.Printf("GetInboxByOwner: %#v\n", key)

	//get all items that relate to passed in owner/address
	var chat []entity.Chatitem
	database.Connector.Where("fromaddr = ?", key).Or("toaddr = ?", key).Find(&chat)

	//get unique conversation addresses
	var uniqueChatMembers []string
	for _, chatitem := range chat {
		//fmt.Printf("search for unique addrs")
		if chatitem.Fromaddr != key {
			if !stringInSlice(chatitem.Fromaddr, uniqueChatMembers) {
				//fmt.Printf("Unique Addr Found: %#v\n", chatitem.Fromaddr)
				uniqueChatMembers = append(uniqueChatMembers, chatitem.Fromaddr)
			}
		}
		if chatitem.Toaddr != key {
			if !stringInSlice(chatitem.Toaddr, uniqueChatMembers) {
				//fmt.Printf("Unique Addr Found: %#v\n", chatitem.Toaddr)
				uniqueChatMembers = append(uniqueChatMembers, chatitem.Toaddr)
			}
		}
	}

	//fmt.Printf("find first message now")
	//for each unique chat member that is not the owner addr, get the latest message
	var userInbox []entity.Chatiteminbox
	for _, chatmember := range uniqueChatMembers {
		var firstItem entity.Chatitem
		var secondItem entity.Chatitem
		var firstItems []entity.Chatitem
		var secondItems []entity.Chatitem
		//fmt.Printf("Unique Chat Addr Check for : %#v\n", chatmember)
		// rowsto, err := database.Connector.DB().Query("SELECT * FROM chatitems WHERE fromaddr = ? AND toaddr = ? ORDER BY id DESC", chatmember, key)
		// if err != nil {
		// 	fmt.Printf("error 1")
		// }
		// for rowsto.Next() {
		// 	rowsto.Scan(&firstItem)
		// }
		// rowsfrom, err := database.Connector.DB().Query("SELECT * FROM chatitems WHERE fromaddr = ? AND toaddr = ? ORDER BY id DESC", key, chatmember)
		// if err != nil {
		// 	fmt.Printf("error 2")
		// }
		// for rowsfrom.Next() {
		// 	rowsfrom.Scan(&secondItem)
		// }

		database.Connector.Where("fromaddr = ?", chatmember).Where("toaddr = ?", key).Order("id desc").Find(&firstItems)
		if len(firstItems) > 0 {
			firstItem = firstItems[0]
		}
		//fmt.Printf("FirstItem : %#v\n", firstItem)
		database.Connector.Where("fromaddr = ?", key).Where("toaddr = ?", chatmember).Order("id desc").Find(&secondItems)
		if len(secondItems) > 0 {
			secondItem = secondItems[0]
		}
		//fmt.Printf("SecondItem : %#v\n", secondItem)

		//add Unread msg count to both first/second items since we don't know which one is newer yet
		var chatCount []entity.Chatitem
		database.Connector.Where("fromaddr = ?", chatmember).Where("toaddr = ?", key).Where("msgread != ?", true).Find(&chatCount)

		//probably a more effecient way, but
		var firstItemWCount entity.Chatiteminbox
		firstItemWCount.Fromaddr = firstItem.Fromaddr
		firstItemWCount.Toaddr = firstItem.Toaddr
		firstItemWCount.Timestamp = firstItem.Timestamp
		firstItemWCount.Msgread = firstItem.Msgread
		firstItemWCount.Message = firstItem.Message
		firstItemWCount.Unreadcnt = len(chatCount)
		var secondItemWCount entity.Chatiteminbox
		secondItemWCount.Fromaddr = secondItem.Fromaddr
		secondItemWCount.Toaddr = secondItem.Toaddr
		secondItemWCount.Timestamp = secondItem.Timestamp
		secondItemWCount.Msgread = secondItem.Msgread
		secondItemWCount.Message = secondItem.Message
		secondItemWCount.Unreadcnt = len(chatCount)

		//pick the most recent message
		if firstItem.Fromaddr != "" {
			if secondItem.Fromaddr == "" {
				userInbox = append(userInbox, firstItemWCount)
			} else {
				layout := "2006-01-02T15:04:05.000Z"
				firstTime, error := time.Parse(layout, firstItem.Timestamp)
				if error != nil {
					//fmt.Println(error)
					return
				}
				secondTime, error := time.Parse(layout, secondItem.Timestamp)
				if error != nil {
					//fmt.Println(error)
					return
				}

				if firstTime.After(secondTime) {
					userInbox = append(userInbox, firstItemWCount)
				} else {
					userInbox = append(userInbox, secondItemWCount)
				}
			}
		} else if secondItem.Fromaddr != "" {
			userInbox = append(userInbox, secondItemWCount)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userInbox)
}

//*********chat info*********************
//GetAllChatitems get all chat data
func GetAllChatitems(w http.ResponseWriter, r *http.Request) {
	var chat []entity.Chatitem
	database.Connector.Find(&chat)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(chat)
}

//Get all unread messages TO a specific user, used for total count notification at top notification bar
func GetUnreadMsgCntTotal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["address"]

	var chat []entity.Chatitem
	database.Connector.Where("toaddr = ?", key).Where("msgread != ?", true).Find(&chat)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(len(chat))
}

func GetUnreadMsgCntNft(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["address"]
	addr := vars["nftaddr"]
	id := vars["nftid"]

	var chat []entity.Chatitem
	database.Connector.Where("toaddr = ?", key).Where("nftaddr = ?", addr).Where("nftid = ?", id).Where("msgread = ?", false).Find(&chat)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(len(chat))
}

//unread count per conversation
func GetUnreadMsgCnt(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	to := vars["toaddr"]
	owner := vars["fromaddr"]

	var chat []entity.Chatitem
	database.Connector.Where("toaddr = ?", to).Where("fromaddr = ?", owner).Where("msgread != ?", true).Find(&chat)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(len(chat))
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

//return both directions of this chat
func GetChatFromAddressToAddr(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	from := vars["fromaddr"]
	to := vars["toaddr"]

	var chat []entity.Chatitem
	database.Connector.Where("fromaddr = ?", from).Where("toaddr = ?", to).Find(&chat)

	var chat2 []entity.Chatitem
	database.Connector.Where("fromaddr = ?", to).Where("toaddr = ?", from).Find(&chat2)

	//chat = append(chat, chat2...)

	//this is aweful but these other commented out ways just are not working
	//var returnChat []entity.Chatitem
	layout := "2006-01-02T15:04:05.000Z"
	//last := "1971-01-02T15:04:05.000Z"
	// lastTime, error := time.Parse(layout, last)
	// if error != nil {
	// 	return
	// }
	for _, chatmember := range chat2 {
		currTime, error := time.Parse(layout, chatmember.Timestamp)
		if error != nil {
			return
		}
		found := false
		//both lists are already sorted, so we can use the assumption here
		for i := 0; i < len(chat); i++ {
			ret_time, error := time.Parse(layout, chat[i].Timestamp)
			if error != nil {
				return
			}
			if currTime.Before(ret_time) {
				chat = append(chat[:i+1], chat[i:]...)
				chat[i] = chatmember
				found = true
				break
			}
		}
		if !found {
			chat = append(chat, chatmember)
		}
	}

	// type NamedArgument struct {
	// 	To   string
	// 	From string
	// }
	//this is bad, shouldn't have to do this but the above complex query is not working for me
	//database.Connector.Raw("select * from chatitems where (fromaddr = @from, AND toaddr = @to) OR (fromaddr = @to AND toaddr = @from)", NamedArgument{To: toaddr, From: fromaddr}).Find(&chat)

	// database.Connector.Where(
	// 	database.Connector.Where("fromaddr = ?", from).Where("toaddr = ?", to),
	// ).Or(
	// 	database.Connector.Where("fromaddr = ?", to).Where("toaddr = ?", from),
	// ).Find(&chat)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chat)
}

func GetChatNftContext(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nftaddr := vars["nftaddr"]
	nftid := vars["nftid"]

	var chat []entity.Chatitem
	database.Connector.Where("nftaddr = ?", nftaddr).Where("nftid = ?", nftid).Find(&chat)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chat)
}

func GetChatNftAllItemsFromAddr(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	from := vars["fromaddr"]
	to := vars["toaddr"]
	addr := vars["nftaddr"]
	id := vars["nftid"]

	var chat []entity.Chatitem
	database.Connector.Where("fromaddr = ?", from).Where("toaddr = ?", to).Where("nftaddr = ?", addr).Where("nftid = ?", id).Find(&chat)

	var chat2 []entity.Chatitem
	database.Connector.Where("fromaddr = ?", to).Where("toaddr = ?", from).Where("nftaddr = ?", addr).Where("nftid = ?", id).Find(&chat2)

	//this is aweful but the complex OR query is just not working in this golang implementation
	//var returnChat []entity.Chatitem
	layout := "2006-01-02T15:04:05.000Z"
	//last := "1971-01-02T15:04:05.000Z"
	// lastTime, error := time.Parse(layout, last)
	// if error != nil {
	// 	return
	// }
	for _, chatmember := range chat2 {
		currTime, error := time.Parse(layout, chatmember.Timestamp)
		if error != nil {
			return
		}
		found := false
		//both lists are already sorted, so we can use the assumption here
		for i := 0; i < len(chat); i++ {
			ret_time, error := time.Parse(layout, chat[i].Timestamp)
			if error != nil {
				return
			}
			if currTime.Before(ret_time) {
				chat = append(chat[:i+1], chat[i:]...)
				chat[i] = chatmember
				found = true
				break
			}
		}
		if !found {
			chat = append(chat, chatmember)
		}
	}

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

	json.Unmarshal(requestBody, &chat)

	//for now only support updating the read status
	//we would need to re-encrypt the data on message update (not hard just need to add it)
	// database.Connector.Model(&entity.Chatitem{}).
	// 	Where("fromaddr = ?", chat.Fromaddr).
	// 	Where("toaddr = ?", chat.Toaddr).
	// 	Where("timestamp = ?", chat.Timestamp).
	// 	Update("message", chat.Message)
	database.Connector.Model(&entity.Chatitem{}).
		Where("fromaddr = ?", chat.Fromaddr).
		Where("toaddr = ?", chat.Toaddr).
		Where("timestamp = ?", chat.Timestamp).
		Update("msgread", chat.Msgread)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(chat)
}

func DeleteAllChatitemsToAddressByOwner(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	to := vars["toaddr"]
	owner := vars["fromaddr"]

	var chat entity.Chatitem

	database.Connector.Where("toaddr = ?", to).Where("fromaddr = ?", owner).Delete(&chat)
	w.WriteHeader(http.StatusNoContent)
}

func CreateSettings(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var settings entity.Settings
	json.Unmarshal(requestBody, &settings)

	database.Connector.Create(settings)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(settings)
}

func UpdateSettings(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var settings entity.Settings

	json.Unmarshal(requestBody, &settings)
	database.Connector.Model(&entity.Settings{}).Where("walletaddr = ?", settings.Walletaddr).Update("publickey", settings.Publickey)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(settings)
}

func DeleteSettings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["address"]

	var settings entity.Settings

	database.Connector.Where("walletaddr = ?", key).Delete(&settings)
	w.WriteHeader(http.StatusNoContent)
}

func GetSettings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["address"]

	var settings []entity.Settings
	database.Connector.Where("walletaddr = ?", key).Find(&settings)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func CreateComments(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var comment entity.Comments
	json.Unmarshal(requestBody, &comment)

	database.Connector.Create(comment)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comment)
}

// func UpdateComments(w http.ResponseWriter, r *http.Request) {
// 	requestBody, _ := ioutil.ReadAll(r.Body)
// 	var comment entity.Comment

// 	json.Unmarshal(requestBody, &comment)
// 	database.Connector.Model(&entity.Settings{}).Where("walletaddr = ?", settings.Walletaddr).Update("publickey", settings.Publickey)

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(comment)
// }

func DeleteComments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fromaddr := vars["address"]
	nftaddr := vars["nftaddr"]
	nftid := vars["nftid"]

	var comment entity.Comments

	database.Connector.Where("fromaddr = ?", fromaddr).Where("nftaddr = ?", nftaddr).Where("nftid = ?", nftid).Delete(&comment)
	w.WriteHeader(http.StatusNoContent)
}

func GetComments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["nftid"]
	addr := vars["nftaddr"]

	var comment []entity.Comments
	database.Connector.Where("nftaddr = ?", addr).Where("nftid = ?", id).Find(&comment)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comment)
}

func GetAllComments(w http.ResponseWriter, r *http.Request) {
	var comment []entity.Comments
	database.Connector.Find(&comment)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(comment)
}
