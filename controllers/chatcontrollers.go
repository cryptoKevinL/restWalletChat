package controllers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"rest-go-demo/auth"
	"rest-go-demo/database"
	"rest-go-demo/entity"
	"strconv"
	"strings"
	"time"

	_ "rest-go-demo/docs"

	"github.com/gorilla/mux"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

type NFTPortOwnerOf struct {
	Response string `json:"response"`
	Nfts     []struct {
		ContractAddress string `json:"contract_address"`
		TokenID         string `json:"token_id"`
		CreatorAddress  string `json:"creator_address"`
	} `json:"nfts"`
	Total        int         `json:"total"`
	Continuation interface{} `json:"continuation"`
}

// GetInboxByOwner godoc
// @Summary Get Inbox Summary With Last Message
// @Description Get Each 1-on-1 Conversation, NFT and Community Chat For Display in Inbox
// @Tags Inbox
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param address path string true "Wallet Address"
// @Success 200 {array} entity.Chatiteminbox
// @Router /v1/get_inbox/{address} [get]
func GetInboxByOwner(w http.ResponseWriter, r *http.Request) {
	//GetInboxByID returns the latest message for each unique conversation
	//vars := mux.Vars(r)
	Authuser := auth.GetUserFromReqContext(r)
	key := Authuser.Address //vars["address"] //owner of the inbox

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
				uniqueChatMembers = append(uniqueChatMembers, chatitem.Fromaddr)
			}
		}
		if chatitem.Toaddr != key {
			if !stringInSlice(chatitem.Toaddr, uniqueChatMembers) {
				uniqueChatMembers = append(uniqueChatMembers, chatitem.Toaddr)
			}
		}
	}

	//fmt.Printf("find first message now")
	//for each unique chat member that is not the owner addr, get the latest message
	var userInbox []entity.Chatiteminbox
	for _, chatmember := range uniqueChatMembers {
		// //add Unread msg count to both first/second items since we don't know which one is newer yet
		var chatCount []entity.Chatitem
		database.Connector.Where("fromaddr = ?", chatmember).Where("toaddr = ?", key).Where("msgread != ?", true).Find(&chatCount)

		// //get name for return val
		var addrname entity.Addrnameitem
		database.Connector.Where("address = ?", chatmember).Find(&addrname)

		//database view - local code replaced 7/14
		var vchatitem entity.V_chatitem
		var dbQuery = database.Connector.Where("fromaddr = ? AND toaddr = ?", key, chatmember).Find(&vchatitem)
		//var dbQuery = database.Connector.Raw("select * from v_chatitems WHERE fromaddr in('0xcafebabe', '0xdeadbeef');").Scan(&testView)

		var itemToInsert entity.Chatiteminbox
		if dbQuery.RowsAffected > 0 {
			itemToInsert.Id = vchatitem.Id
			itemToInsert.Fromaddr = vchatitem.Fromaddr
			itemToInsert.Toaddr = vchatitem.Toaddr
			itemToInsert.Timestamp = vchatitem.Timestamp
			itemToInsert.Timestamp_dtm = vchatitem.Timestamp_dtm
			itemToInsert.Msgread = vchatitem.Msgread
			itemToInsert.Message = vchatitem.Message
			itemToInsert.Unreadcnt = len(chatCount)
			itemToInsert.Contexttype = entity.DM
			itemToInsert.Type = entity.Message
			itemToInsert.Sendername = addrname.Name
			itemToInsert.Encryptsymkey = vchatitem.Encryptsymkey
			itemToInsert.Litaccesscond = vchatitem.Litaccesscond
			//fmt.Printf("encrypted symmetric LIT key: %#v %#v %#v\n", vchatitem.Encryptsymkey, vchatitem.Toaddr, vchatitem.Fromaddr)

			found := false
			for i := 0; i < len(userInbox); i++ {
				if itemToInsert.Timestamp_dtm.After(userInbox[i].Timestamp_dtm) {
					userInbox = append(userInbox[:i+1], userInbox[i:]...)
					userInbox[i] = itemToInsert
					found = true
					break
				}
			}
			if !found {
				userInbox = append(userInbox, itemToInsert)
			}
			//end timesort the append
		}
	}

	//now get bookmarked/joined groups as well but fit it into the inbox return val type
	var bookmarks []entity.Bookmarkitem
	database.Connector.Where("walletaddr = ?", key).Find(&bookmarks)

	//TODO: need to throttle these 2 calls to auto-join?
	//should auto-join them to the community chat
	AutoJoinCommunitiesByChain(key, "ethereum")
	AutoJoinCommunitiesByChain(key, "polygon")
	AutoJoinPoapChats(key)

	//now add last message from group chat this bookmark is for
	var gchat []entity.Groupchatitem //even though I use this in a Last() function I need to store as an array, or subsequenct DB queries fail!
	for idx := 0; idx < len(bookmarks); idx++ {
		//fmt.Printf("bookmarks: %#v\n", bookmarks[i])
		//fmt.Printf("\nnftaddr: %#v\n", bookmarks[idx].Nftaddr)
		dbQuery := database.Connector.Where("nftaddr = ?", bookmarks[idx].Nftaddr).Last(&gchat)
		//fmt.Printf("dbQuery: %#v\n", dbQuery.Error)

		var returnItem entity.Chatiteminbox
		if dbQuery.RowsAffected == 0 {
			//if this chat is new/empty just return the basic info
			returnItem.Nftaddr = bookmarks[idx].Nftaddr
			returnItem.Contexttype = entity.Community
			if strings.HasPrefix(returnItem.Nftaddr, "0x") {
				returnItem.Contexttype = entity.Nft
				returnItem.Chain = bookmarks[idx].Chain
			}
			if strings.HasPrefix(returnItem.Nftaddr, "poap_") {
				returnItem.Contexttype = entity.Nft
				returnItem.Chain = bookmarks[idx].Chain
			}
			userInbox = append(userInbox, returnItem)
			continue
		}
		//fmt.Printf("bookmarkchat: %#v\n", gchat)

		var groupchat = gchat[0]

		//get num unread messages
		var chatCnt []entity.Groupchatitem
		var chatReadTime entity.Groupchatreadtime
		dbQuery = database.Connector.Where("fromaddr = ?", key).Where("nftaddr = ?", groupchat.Nftaddr).Find(&chatReadTime)
		//if no respsonse to this query, its the first time a user is reading the chat history, send it all
		if dbQuery.RowsAffected == 0 {
			//fmt.Printf("sending all values! \n")
			database.Connector.Where("nftaddr = ?", groupchat.Nftaddr).Find(&chatCnt)
		} else {
			database.Connector.Where("timestamp_dtm > ?", chatReadTime.Readtimestamp_dtm).Where("nftaddr = ?", groupchat.Nftaddr).Find(&chatCnt)
			//fmt.Printf("sending time based count \n")
		}
		//end get num unread messages

		returnItem.Id = groupchat.Id
		returnItem.Message = groupchat.Message
		returnItem.Timestamp = groupchat.Timestamp
		returnItem.Timestamp_dtm = groupchat.Timestamp_dtm
		returnItem.Nftaddr = groupchat.Nftaddr
		returnItem.Fromaddr = groupchat.Fromaddr
		returnItem.Unreadcnt = len(chatCnt)
		returnItem.Type = groupchat.Type
		returnItem.Chain = bookmarks[idx].Chain
		//retrofit old messages prior to setting Type
		if returnItem.Type != entity.Message && returnItem.Type != entity.Welcome {
			returnItem.Type = entity.Message
		}
		returnItem.Contexttype = entity.Community

		//get common name from nftaddress
		var addrname entity.Addrnameitem
		var result = database.Connector.Where("address = ?", groupchat.Nftaddr).Find(&addrname)
		if result.RowsAffected > 0 {
			returnItem.Name = addrname.Name
		}
		//not sure if long term we will store by name (WalletChat HQ) or nftaddr (walletchat)
		var imgname entity.Imageitem
		result = database.Connector.Where("name = ?", returnItem.Name).Find(&imgname)
		if result.RowsAffected > 0 {
			returnItem.LogoData = imgname.Base64data
		}

		//until we fix up old tables, we can hack this to double check
		if strings.HasPrefix(returnItem.Nftaddr, "0x") {
			returnItem.Contexttype = entity.Nft
		}
		if strings.HasPrefix(returnItem.Nftaddr, "poap_") {
			returnItem.Contexttype = entity.Nft
		}

		returnItem.Sendername = ""
		if returnItem.Message == "" {
			var unsetTime time.Time
			var noInt int
			returnItem.Unreadcnt = noInt
			returnItem.Timestamp = unsetTime.String()
		}

		//timesort the append
		found := false
		for i := 0; i < len(userInbox); i++ {
			if returnItem.Timestamp_dtm.After(userInbox[i].Timestamp_dtm) {
				userInbox = append(userInbox[:i+1], userInbox[i:]...)
				userInbox[i] = returnItem
				found = true
				break
			}
		}
		if !found {
			userInbox = append(userInbox, returnItem)
		}
		//userInbox = append(userInbox, returnItem)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userInbox)
}

//removed since this will take FOREVER and its not used
// func GetAllChatitems(w http.ResponseWriter, r *http.Request) {
// 	var chat []entity.Chatitem
// 	database.Connector.Find(&chat)

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(chat)
// }

// GetUnreadMsgCntTotal godoc
// @Summary Get all unread messages TO a specific user, used for total count notification at top notification bar
// @Description Get Each 1-on-1 Conversation, NFT and Community Chat For Display in Inbox
// @Tags Inbox
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param address path string true "Wallet Address"
// @Success 200 {integer} int
// @Router /v1/get_unread_cnt/{address} [get]
func GetUnreadMsgCntTotal(w http.ResponseWriter, r *http.Request) {
	//vars := mux.Vars(r)
	Authuser := auth.GetUserFromReqContext(r)
	key := Authuser.Address //vars["address"]

	var chat []entity.Chatitem
	database.Connector.Where("toaddr = ?", key).Where("msgread != ?", true).Find(&chat)

	//get group chat unread items as well

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(len(chat))
}

// GetUnreadMsgCntTotalByType godoc
// @Summary Get all unread messages TO a specific user, used for total count notification at top notification bar
// @Description Get Each 1-on-1 Conversation, NFT and Community Chat For Display in Inbox
// @Tags Unused/Legacy
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param address path string true "Wallet Address"
// @Param type path string true "Message Type - nft|community|dm|all"
// @Success 200 {integer} int
// @Router /v1/get_unread_cnt_by_type/{address}/{type} [get]
func GetUnreadMsgCntTotalByType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	Authuser := auth.GetUserFromReqContext(r)
	key := Authuser.Address //key := vars["address"]
	msgtype := vars["type"] //nft/community/DM/ALL

	msgCntTotal := 0

	var bookmarks []entity.Bookmarkitem
	database.Connector.Where("walletaddr = ?", key).Find(&bookmarks)

	//now add last message from group chat this bookmark is for
	var gchat []entity.Groupchatitem //even though I use this in a Last() function I need to store as an array, or subsequenct DB queries fail!
	if msgtype == entity.Nft || msgtype == entity.Community || msgtype == entity.All {
		for idx := 0; idx < len(bookmarks); idx++ {
			dbQuery := database.Connector.Where("nftaddr = ?", bookmarks[idx].Nftaddr).Last(&gchat)
			if dbQuery.RowsAffected == 0 {
				continue
			}
			var groupchat = gchat[0]

			//get num unread messages
			var chatCnt []entity.Groupchatitem
			var chatReadTime entity.Groupchatreadtime
			dbQuery = database.Connector.Where("fromaddr = ?", key).Where("nftaddr = ?", groupchat.Nftaddr).Find(&chatReadTime)
			//if no respsonse to this query, its the first time a user is reading the chat history
			if dbQuery.RowsAffected == 0 {
				database.Connector.Where("nftaddr = ?", groupchat.Nftaddr).Find(&chatCnt)
			} else {
				database.Connector.Where("timestamp_dtm > ?", chatReadTime.Readtimestamp_dtm).Where("nftaddr = ?", groupchat.Nftaddr).Find(&chatCnt)
			}
			//end get num unread messages

			if strings.HasPrefix(groupchat.Nftaddr, "0x") {
				if msgtype == entity.Nft || msgtype == entity.All {
					msgCntTotal += len(chatCnt)
				}
			} else if msgtype == entity.Community || msgtype == entity.All {
				msgCntTotal += len(chatCnt)
			}
		}
	}
}

// func PutUnreadcnt(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	walletaddr := vars["address"]

// 	requestBody, _ := ioutil.ReadAll(r.Body)
// 	var config entity.Unreadcountitem
// 	json.Unmarshal(requestBody, &config)

// 	var findConfig entity.Unreadcountitem
// 	var dbQuery = database.Connector.Where("walletaddr = ?", walletaddr).Find(&findConfig)

// 	if dbQuery.RowsAffected == 0 {
// 		config.Walletaddr = walletaddr
// 		database.Connector.Create(&config)
// 	} else {
// 		database.Connector.Model(&entity.Unreadcountitem{}).Where("walletaddr = ?", walletaddr).Update("dm", config.Dm)
// 		database.Connector.Model(&entity.Unreadcountitem{}).Where("walletaddr = ?", walletaddr).Update("nft", config.Nft)
// 		database.Connector.Model(&entity.Unreadcountitem{}).Where("walletaddr = ?", walletaddr).Update("community", config.Community)
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusCreated)
// 	json.NewEncoder(w).Encode(true)
// }

// GetUnreadcnt godoc
// @Summary Get all unread messages TO a specific user, used for total count notification at top notification bar
// @Description Get Unread count just given an address
// @Tags Inbox
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param address path string true "Wallet Address"
// @Success 200 {integer} int
// @Router /v1/unreadcount/{address} [get]
func GetUnreadcnt(w http.ResponseWriter, r *http.Request) {
	//vars := mux.Vars(r)
	Authuser := auth.GetUserFromReqContext(r)
	key := Authuser.Address
	//key := vars["address"]

	//get configured items from DB
	var config entity.Unreadcountitem
	// var dbQuery = database.Connector.Where("walletaddr = ?", key).Find(&config)

	// if dbQuery.RowsAffected == 0 {
	// 	//create a config //this can be removed eventually once all accounts have a saved setting
	// 	config.Community = 0
	// 	config.Dm = 0
	// 	config.Nft = 0
	// 	config.Walletaddr = key
	// 	database.Connector.Create(&config)
	// }

	var bookmarks []entity.Bookmarkitem
	database.Connector.Where("walletaddr = ?", key).Find(&bookmarks)

	//now add last message from group chat this bookmark is for
	var gchat []entity.Groupchatitem //even though I use this in a Last() function I need to store as an array, or subsequenct DB queries fail!
	for idx := 0; idx < len(bookmarks); idx++ {
		dbQuery := database.Connector.Where("nftaddr = ?", bookmarks[idx].Nftaddr).Last(&gchat)
		if dbQuery.RowsAffected == 0 {
			continue
		}
		var groupchat = gchat[0]

		//get num unread messages
		var chatCnt []entity.Groupchatitem
		var chatReadTime entity.Groupchatreadtime
		dbQuery = database.Connector.Where("fromaddr = ?", key).Where("nftaddr = ?", groupchat.Nftaddr).Find(&chatReadTime)
		//if no respsonse to this query, its the first time a user is reading the chat history
		if dbQuery.RowsAffected == 0 {
			database.Connector.Where("nftaddr = ?", groupchat.Nftaddr).Find(&chatCnt)
		} else {
			database.Connector.Where("timestamp_dtm > ?", chatReadTime.Readtimestamp_dtm).Where("nftaddr = ?", groupchat.Nftaddr).Find(&chatCnt)
		}
		//end get num unread messages

		if strings.HasPrefix(groupchat.Nftaddr, "0x") {
			config.Nft += len(chatCnt)
		} else {
			config.Community += len(chatCnt)
		}
	}

	var chat []entity.Chatitem
	database.Connector.Where("toaddr = ?", key).Where("msgread != ?", true).Find(&chat)
	config.Dm = len(chat)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(config)
}

// GetUnreadMsgCntNft godoc
// @Summary Get all unread messages for a specific NFT context
// @Description Get Unread count for specifc NFT context given a wallet address and specific NFT
// @Tags NFT
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param address path string true "Wallet Address"
// @Param nftaddr path string true "NFT Contract Address"
// @Param nftid path string true "NFT ID"
// @Success 200 {integer} int
// @Router /v1/get_unread_cnt/{address}/{nftaddr}/{nftid} [get]
func GetUnreadMsgCntNft(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	Authuser := auth.GetUserFromReqContext(r)
	key := Authuser.Address
	//key := vars["address"]
	addr := vars["nftaddr"]
	id := vars["nftid"]

	var chat []entity.Chatitem
	database.Connector.Where("toaddr = ?", key).Where("nftaddr = ?", addr).Where("nftid = ?", id).Where("msgread = ?", false).Find(&chat)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(len(chat))
}

// GetUnreadMsgCntNft godoc
// @Summary Get all unread messages for all NFT related chats for given user
// @Description Get Unread count for all NFT contexts given a wallet address
// @Tags Unused/Legacy
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param address path string true "Wallet Address"
// @Success 200 {integer} int
// @Router /v1/get_unread_cnt_nft/{address} [get]
func GetUnreadMsgCntNftAllByAddr(w http.ResponseWriter, r *http.Request) {
	// vars := mux.Vars(r)
	// key := vars["address"]
	Authuser := auth.GetUserFromReqContext(r)
	key := Authuser.Address

	var chat []entity.Chatitem
	database.Connector.Where("toaddr = ?", key).Where("nftid != ?", 0).Find(&chat)

	//first we need to find unique senders (has to be a better way to use SQL db for this)
	var senderlist []string
	for i := 0; i < len(chat); i++ {
		if !stringInSlice(chat[i].Fromaddr, senderlist) {
			fmt.Printf("Found Unique Sender: %#v\n", chat[i].Fromaddr)
			senderlist = append(senderlist, chat[i].Fromaddr)
		}
	}

	//now for each sender we need get unique nft contract addresses
	var nftretval []entity.Nftsidebar

	for i := 0; i < len(senderlist); i++ {
		var senderAddr = senderlist[i]
		var chatUniqueNft []entity.Chatitem
		database.Connector.Where("toaddr = ?", key).Where("nftid != ?", 0).Where("fromaddr = ?", senderAddr).Find(&chatUniqueNft)

		var uniquecontracts []string
		for j := 0; j < len(chatUniqueNft); j++ {
			if !stringInSlice(chatUniqueNft[i].Nftaddr, uniquecontracts) {
				fmt.Printf("Found Unique NFT Contract: %#v\n", chatUniqueNft[i].Nftaddr)
				//for the given senderAddr this is unique list of contract addresses
				uniquecontracts = append(uniquecontracts, chatUniqueNft[i].Nftaddr)
			}
		}

		//now for each unqiue sender, and unique nft contract address, get unique NFT ids
		for k := 0; k < len(uniquecontracts); k++ {
			var uniqueNftAddr = uniquecontracts[k]
			var chatUniqueNftIds []entity.Chatitem
			database.Connector.Where("toaddr = ?", key).Where("nftid != ?", 0).Where("fromaddr = ?", senderAddr).Where("nftaddr = ?", uniqueNftAddr).Find(&chatUniqueNftIds)

			var uniquenftids []string
			for l := 0; l < len(chatUniqueNftIds); l++ {
				var nftid = chatUniqueNftIds[l].Nftid
				var chatNftId []entity.Chatitem
				fmt.Printf("Unique NFT ID : %#v\n", nftid)

				database.Connector.Where("toaddr = ?", key).
					Where("nftid = ?", nftid).Where("fromaddr = ?", senderAddr).
					Where("nftaddr = ?", uniqueNftAddr).
					Where("msgread = ?", false).Find(&chatNftId)

				if !stringInSlice(nftid, uniquenftids) {
					uniquenftids = append(uniquenftids, nftid)

					var sbitem entity.Nftsidebar
					sbitem.Fromaddr = senderAddr
					sbitem.Nftaddr = uniqueNftAddr
					sbitem.Nftid = nftid
					sbitem.Unread = len(chatNftId)

					nftretval = append(nftretval, sbitem)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(nftretval)
}

// GetUnreadMsgCnt godoc
// @Summary Get all unread messages between two addresses
// @Description Get Unread count for DMs
// @Tags Unused/Legacy
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param toaddr path string true "TO: Wallet Address"
// @Param from path string true "FROM: Wallet Address"
// @Success 200 {integer} int
// @Router /v1/get_unread_cnt/{fromaddr}/{toaddr} [get]
func GetUnreadMsgCnt(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	//to := vars["toaddr"]
	Authuser := auth.GetUserFromReqContext(r)
	to := Authuser.Address
	owner := vars["fromaddr"]

	var chat []entity.Chatitem
	database.Connector.Where("toaddr = ?", to).Where("fromaddr = ?", owner).Where("msgread != ?", true).Find(&chat)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(len(chat))
}

// GetChatFromAddress godoc
// @Summary Get Chat Item For Given Wallet Address
// @Description Get all Chat Items for DMs for a given wallet address
// @Tags Unused/Legacy
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param toaddr path string true "Wallet Address"
// @Success 200 {array} entity.Chatitem
// @Router /v1/getall_chatitems/{address} [get]
func GetChatFromAddress(w http.ResponseWriter, r *http.Request) {
	// vars := mux.Vars(r)
	// key := vars["address"]
	Authuser := auth.GetUserFromReqContext(r)
	key := Authuser.Address

	var chat []entity.Chatitem
	database.Connector.Where("fromaddr = ?", key).Or("toaddr = ?", key).Find(&chat)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chat)
}

// GetNftChatFromAddress godoc
// @Summary Get NFT Related Chat Items For Given Wallet Address
// @Description Get ALL NFT context items for a given wallet address
// @Tags Unused/Legacy
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param toaddr path string true "Wallet Address"
// @Success 200 {array} entity.Chatitem
// @Router /v1/getnft_chatitems/{address} [get]
func GetNftChatFromAddress(w http.ResponseWriter, r *http.Request) {
	// vars := mux.Vars(r)
	// key := vars["address"]
	Authuser := auth.GetUserFromReqContext(r)
	key := Authuser.Address

	var chat []entity.Chatitem
	database.Connector.Where("fromaddr = ?", key).Where("nftid != ?", 0).Or("toaddr = ?", key).Where("nftid != ?", 0).Find(&chat)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chat)
}

// GetChatFromAddressToAddr godoc
// @Summary Get Chat Data Between Two Addresses
// @Description Get chat data between the given two addresses, TO and FROM and interchangable here
// @Tags DMs
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param toaddr path string true "TO: Wallet Address"
// @Param from path string true "FROM: Wallet Address"
// @Success 200 {array} entity.Chatitem
// @Router /v1/getall_chatitems/{fromaddr}/{toaddr} [get]
func GetAllChatFromAddressToAddr(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	//from := vars["fromaddr"]
	to := vars["toaddr"]
	Authuser := auth.GetUserFromReqContext(r)
	from := Authuser.Address

	var chat []entity.Chatitem
	database.Connector.Where("fromaddr = ?", from).Where("toaddr = ?", to).Find(&chat)

	var chat2 []entity.Chatitem
	database.Connector.Where("fromaddr = ?", to).Where("toaddr = ?", from).Find(&chat2)

	for _, chatmember := range chat2 {
		currTime := chatmember.Timestamp_dtm
		found := false
		//both lists are already sorted, so we can use the assumption here
		for i := 0; i < len(chat); i++ {
			ret_time := chat[i].Timestamp_dtm
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

// GetChatFromAddressToAddr godoc
// @Summary Get Chat Data Between Two Addresses
// @Description Get chat data between the given two addresses, TO and FROM and interchangable here
// @Tags DMs
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param toaddr path string true "TO: Wallet Address"
// @Param from path string true "FROM: Wallet Address"
// @Param time path string true "Load only messages after this time"
// @Success 200 {array} entity.Chatitem
// @Router /v1/getall_chatitems/{fromaddr}/{toaddr}/${time} [get]
func GetNewChatFromAddressToAddr(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	//from := vars["fromaddr"]
	to := vars["toaddr"]
	timeStamp := vars["time"]
	Authuser := auth.GetUserFromReqContext(r)
	from := Authuser.Address

	decodedStr, err := url.QueryUnescape(timeStamp)
	if err != nil {
		fmt.Printf("Error decoding the string %v\r\n", err)
	}
	//fmt.Printf("Input Timestamp: %v", decodedStr)

	layout := "2006-01-02T15:04:05.000Z"
	formattedTime, err := time.Parse(layout, decodedStr)
	if err != nil {
		log.Println(err)
	}
	//fmt.Printf("Formatted Timestamp: %v", formattedTime.Add(time.Second))

	//add a second to timestamp sent in, because conversion rounds up sometimes
	formattedTime = formattedTime.Add(time.Second)
	var chat []entity.Chatitem
	database.Connector.
		Where("fromaddr = ?", from).
		Where("toaddr = ?", to).
		Where("timestamp_dtm > ?", formattedTime).
		Find(&chat)

	var chat2 []entity.Chatitem
	database.Connector.
		Where("fromaddr = ?", to).
		Where("toaddr = ?", from).
		Where("timestamp_dtm > ?", formattedTime).
		Find(&chat2)

	for _, chatmember := range chat2 {
		currTime := chatmember.Timestamp_dtm
		found := false
		//both lists are already sorted, so we can use the assumption here
		for i := 0; i < len(chat); i++ {
			ret_time := chat[i].Timestamp_dtm
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

// GetChatNftContext godoc
// @Summary Get NFT Related Chat Items For Given NFT Contract and ID
// @Description Get ALL NFT context items for a given wallet address
// @Tags Unused/Legacy
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param nftaddr path string true "NFT Contract Address"
// @Param nftid path string true "NFT ID"
// @Success 200 {array} entity.Chatitem
// @Router /v1/getnft_chatitems/{nftaddr}/{nftid} [get]
func GetChatNftContext(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nftaddr := vars["nftaddr"]
	nftid := vars["nftid"]

	var chat []entity.Chatitem
	database.Connector.Where("nftaddr = ?", nftaddr).Where("nftid = ?", nftid).Find(&chat)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chat)
}

// GetChatNftContext godoc
// @Summary Get NFT Related Chat Items For Given NFT Contract and ID, between two wallet addresses (TO and FROM are interchangable)
// @Description Get ALL NFT context items for a specifc NFT context convo between two wallets
// @Tags NFT
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param nftaddr path string true "NFT Contract Address"
// @Param nftid path string true "NFT ID"
// @Param toaddr path string true "TO: Wallet Address"
// @Param from path string true "FROM: Wallet Address"
// @Success 200 {array} entity.Chatitem
// @Router /v1/getnft_chatitems/{fromaddr}/{toaddr}/{nftaddr}/{nftid} [get]
func GetChatNftAllItemsFromAddrAndNFT(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	from := vars["fromaddr"]

	Authuser := auth.GetUserFromReqContext(r)
	to := Authuser.Address
	//to := vars["toaddr"]
	addr := vars["nftaddr"]
	id := vars["nftid"]

	var chat []entity.Chatitem
	database.Connector.Where("fromaddr = ?", from).Where("toaddr = ?", to).Where("nftaddr = ?", addr).Where("nftid = ?", id).Find(&chat)
	//fmt.Printf("Chat Items: %#v\n", chat)

	var chat2 []entity.Chatitem
	database.Connector.Where("fromaddr = ?", to).Where("toaddr = ?", from).Where("nftaddr = ?", addr).Where("nftid = ?", id).Find(&chat2)
	//fmt.Printf("Chat2 Items: %#v\n", chat2)

	//TODO: should be a way to called a stored proc for this to sort in MySQL using timestamp
	for _, chatmember := range chat2 {
		currTime := chatmember.Timestamp_dtm
		found := false
		//both lists are already sorted, so we can use the assumption here
		for i := 0; i < len(chat); i++ {
			ret_time := chat[i].Timestamp_dtm

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

// GetChatNftAllItemsFromAddr godoc
// @Summary Get NFT Related Chat Items For Given NFT Contract and ID, relating to one wallet
// @Description Get all specified NFT contract and ID items for a given wallet address
// @Tags Unused/Legacy
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param address path string true "Wallet Address"
// @Param nftaddr path string true "NFT Contract Address"
// @Param nftid path string true "NFT ID"
// @Success 200 {array} entity.Chatitem
// @Router /v1/getnft_chatitems/{address}/{nftaddr}/{nftid} [get]
func GetChatNftAllItemsFromAddr(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	//walletaddr := vars["address"]
	Authuser := auth.GetUserFromReqContext(r)
	walletaddr := Authuser.Address
	addr := vars["nftaddr"]
	id := vars["nftid"]

	var chat []entity.Chatitem
	database.Connector.Where("fromaddr = ?", walletaddr).Where("nftaddr = ?", addr).Where("nftid = ?", id).Find(&chat)

	var chat2 []entity.Chatitem
	database.Connector.Where("toaddr = ?", walletaddr).Where("nftaddr = ?", addr).Where("nftid = ?", id).Find(&chat2)

	//TODO, should do this and similar sorts in a stored proc probably which sort (call 2 queries above with and ORDER)
	for _, chatmember := range chat2 {
		currTime := chatmember.Timestamp_dtm
		found := false
		//both lists are already sorted, so we can use the assumption here
		for i := 0; i < len(chat); i++ {
			ret_time := chat[i].Timestamp_dtm
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

// CreateChatitem godoc
// @Summary Create/Insert DM Chat Message (1-to-1 messaging)
// @Description For DMs, Chatitem data struct is used to store each message and associated info.
// @Description REQUIRED: fromaddr, toaddr, message (see data struct section at bottom of page for more detailed info on each paramter)
// @Description Other fields are generally filled in by the backed REST API and used as return parameters
// @Description ID is auto generated and should never be used as input.
// @Tags DMs
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param message body entity.Chatitem true "Direct Message Chat Data"
// @Success 200 {array} entity.Chatitem
// @Router /v1/create_chatitem [post]
func CreateChatitem(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var chat entity.Chatitem
	json.Unmarshal(requestBody, &chat)

	//added this because from API doc it was throwing error w/o this
	//TODO: we should sort out if we really need this as an input or output only
	chat.Timestamp = time.Now().Format("2006-01-02T15:04:05.000Z")
	//I think can remove this too since Oliver added a DB trigger
	chat.Timestamp_dtm = time.Now()

	//ensure user in body is same as user in JWT
	Authuser := auth.GetUserFromReqContext(r)
	walletaddr := Authuser.Address
	chat.Fromaddr = strings.ToLower(chat.Fromaddr)

	if walletaddr == chat.Fromaddr {
		dbQuery := database.Connector.Create(&chat)
		if dbQuery.RowsAffected == 0 {
			fmt.Println(dbQuery.Error)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(dbQuery.Error)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(chat)

			//also notify the TO user of a new message (need to throttle this somehow)
			var settings entity.Settings
			var dbResult = database.Connector.Where("walletaddr = ?", chat.Toaddr).Find(&settings)
			if dbResult.RowsAffected > 0 {
				if strings.Contains(settings.Email, "@") {
					var fromAddrname entity.Addrnameitem
					database.Connector.Where("address = ?", chat.Fromaddr).Find(&fromAddrname)
					var toAddrname entity.Addrnameitem
					database.Connector.Where("address = ?", chat.Toaddr).Find(&toAddrname)

					from := mail.NewEmail("WalletChat Notifications", "contact@walletchat.fun")
					subject := "Message Waiting In WalletChat"
					to := mail.NewEmail(toAddrname.Name, settings.Email)
					plainTextContent := "You have message from" + fromAddrname.Name + " waiting in WalletChat, please login via the app direct to read!"
					htmlContent := "<strong>You have message from " + fromAddrname.Name + " waiting in WalletChat, please login via the app direct to read!</strong>"
					message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
					client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
					response, err := client.Send(message)
					if err != nil {
						log.Println(err)
					} else {
						fmt.Println(response.StatusCode)
						fmt.Println(response.Body)
						fmt.Println(response.Headers)
					}
				}
			}
		}
	} else {
		fmt.Println("create_chatitem - JWT Address: ", Authuser.Address)
		fmt.Println("create_chatitem - POST Address: ", chat.Fromaddr)
		w.WriteHeader(http.StatusForbidden)
	}
}

// CreateGroupChatitem godoc
// @Summary Create/Insert chat message for Community/NFT/Group Messaging
// @Description Currently used for all messages outside of DMs
// @Tags GroupChat
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param message body entity.Groupchatitem true "Group Message Chat Data"
// @Success 200 {array} entity.Groupchatitem
// @Router /v1/create_groupchatitem [post]
func CreateGroupChatitem(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var chat entity.Groupchatitem
	json.Unmarshal(requestBody, &chat)

	//these will get overwritten as needed when returning data
	chat.Contexttype = entity.Nft
	chat.Type = entity.Message

	//probably can removed now with DB trigger
	chat.Timestamp_dtm = time.Now()

	Authuser := auth.GetUserFromReqContext(r)

	if chat.Fromaddr == Authuser.Address {
		database.Connector.Create(&chat)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(chat)
	} else {
		w.WriteHeader(http.StatusForbidden)
	}
}

// CreateCommunityChatitem godoc
// @Summary CreateCommunityChatitem creates GroupChatitem just with community tag (likely could be consolidated)
// @Description Community Chat Data
// @Tags GroupChat
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param message body entity.Groupchatitem true "Community Message Chat Data"
// @Success 200 {array} entity.Groupchatitem
// @Router /v1/community [post]
func CreateCommunityChatitem(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var chat entity.Groupchatitem
	if err := json.Unmarshal(requestBody, &chat); err != nil { // Parse []byte to the go struct pointer
		fmt.Println("Can not unmarshal JSON in CreateCommunityChat")
	}

	//set type (could hack this in GET side but this is probably cleaner?)
	if chat.Type != entity.Welcome {
		chat.Type = entity.Message
	}

	//can remove now I think
	chat.Timestamp_dtm = time.Now()

	Authuser := auth.GetUserFromReqContext(r)
	if chat.Fromaddr == Authuser.Address {
		database.Connector.Create(&chat)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(chat)
	} else {
		w.WriteHeader(http.StatusForbidden)
	}
}

// GetGroupChatItems godoc
// @Summary GetGroupChatItems gets group chat data for a given NFT address
// @Description Community Chat Data
// @Tags NFT
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param message path string true "Get Group Chat Data By NFT Address"
// @Success 200 {array} entity.Groupchatitem
// @Router /v1/get_groupchatitems/{address} [get]
func GetGroupChatItems(w http.ResponseWriter, r *http.Request) {
	//vars := mux.Vars(r)
	//key := vars["address"]
	Authuser := auth.GetUserFromReqContext(r)
	key := Authuser.Address

	var chat []entity.Groupchatitem
	database.Connector.Where("nftaddr = ?", key).Find(&chat)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chat)
}

// CreateBookmarkItem godoc
// @Summary Join an NFT or Community group chat
// @Description Bookmarks keep an NFT/Community group chat in the sidebar
// @Tags GroupChat
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param message body entity.Bookmarkitem true "Add Bookmark from Community Group Chat"
// @Success 200 {array} entity.Bookmarkitem
// @Router /v1/create_bookmark [post]
func CreateBookmarkItem(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var bookmark entity.Bookmarkitem
	json.Unmarshal(requestBody, &bookmark)

	Authuser := auth.GetUserFromReqContext(r)

	if bookmark.Walletaddr == Authuser.Address {
		//fmt.Printf("Bookmark Item: %#v\n", chat)
		bookmark.Chain = "none"
		//0x check is a cheap hack right now since NFTPort.xyz is rate limiting us a lot
		if strings.HasPrefix(bookmark.Nftaddr, "0x") {
			bookmark.Chain = "ethereum"
		}
		if strings.HasPrefix(bookmark.Nftaddr, "poap_") {
			bookmark.Chain = "xdai"
		}
		//end hack for limiting NFTport API
		var result = IsOnChain(bookmark.Nftaddr, "ethereum")
		if result {
			bookmark.Chain = "ethereum"
		} else {
			var result = IsOnChain(bookmark.Nftaddr, "polygon")
			if result {
				bookmark.Chain = "polygon"
			}
		}

		database.Connector.Create(&bookmark)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(bookmark)
	} else {
		w.WriteHeader(http.StatusForbidden)
	}
}

// DeleteBookmarkItem godoc
// @Summary Leave an NFT or Community group chat
// @Description Bookmarks keep an NFT/Community group chat in the sidebar
// @Tags GroupChat
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param message body entity.Bookmarkitem true "Remove Bookmark from Community Group Chat"
// @Success 200 {array} entity.Bookmarkitem
// @Router /v1/delete_bookmark [post]
func DeleteBookmarkItem(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var bookmark entity.Bookmarkitem
	json.Unmarshal(requestBody, &bookmark)

	Authuser := auth.GetUserFromReqContext(r)

	if bookmark.Walletaddr == Authuser.Address {
		var success = database.Connector.Where("nftaddr = ?", bookmark.Nftaddr).Where("walletaddr = ?", bookmark.Walletaddr).Delete(bookmark)

		var returnval bool
		if success.RowsAffected > 0 {
			returnval = true
		}

		//set the fact the user has manually unjoined this NFT
		var tempUserUnjoined entity.Userunjoined
		var checkUser = database.Connector.Where("nftaddr = ?", bookmark.Nftaddr).Where("walletaddr = ?", bookmark.Walletaddr).Find(&tempUserUnjoined)

		if checkUser.RowsAffected > 0 {
			database.Connector.Model(&entity.Userunjoined{}).
				Where("walletaddr = ?", bookmark.Walletaddr).
				Where("nftaddr = ?", bookmark.Nftaddr).
				Update("unjoined", true)
		} else {
			tempUserUnjoined.Nftaddr = bookmark.Nftaddr
			tempUserUnjoined.Walletaddr = bookmark.Walletaddr
			tempUserUnjoined.Unjoined = true
			database.Connector.Create(&tempUserUnjoined)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(returnval)
	} else {
		w.WriteHeader(http.StatusForbidden)
	}
}

// IsBookmarkItem godoc
// @Summary Check if a wallet address has bookmarked/joined given NFT contract
// @Description This used for UI purposes, checking if a user/wallet has bookmarked a community.
// @Tags GroupChat
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param walletaddr path string true "Wallet Address"
// @Param nftaddr path string true "NFT Contract Address"
// @Success 200 {bool} bool
// @Router /v1/get_bookmarks/{walletaddr}/{nftaddr} [get]
func IsBookmarkItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	//walletaddr := vars["walletaddr"]
	nftaddr := vars["nftaddr"]

	Authuser := auth.GetUserFromReqContext(r)
	walletaddr := Authuser.Address

	var bookmarks []entity.Bookmarkitem
	database.Connector.Where("walletaddr = ?", walletaddr).Where("nftaddr = ?", nftaddr).Find(&bookmarks)

	var returnval bool
	if len(bookmarks) > 0 {
		returnval = true
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(returnval)
}

// GetBookmarkItems godoc
// @Summary Check if a wallet address has bookmarked/joined given NFT contract
// @Description This used for UI purposes, checking if a user/wallet has bookmarked a community.
// @Tags GroupChat
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param address path string true "Wallet Address"
// @Success 200 {array} entity.Bookmarkitem
// @Router /v1/get_bookmarks/{address}/ [get]
func GetBookmarkItems(w http.ResponseWriter, r *http.Request) {
	// vars := mux.Vars(r)
	// key := vars["address"]
	Authuser := auth.GetUserFromReqContext(r)
	key := Authuser.Address

	var bookmarks []entity.Bookmarkitem
	database.Connector.Where("walletaddr = ?", key).Find(&bookmarks)

	//now add last message from group chat this bookmark is for
	var returnItems []entity.BookmarkReturnItem
	var chat entity.Groupchatitem
	for i := 0; i < len(bookmarks); i++ {
		chat.Message = ""
		//chat.Timestamp
		database.Connector.Where("nftaddr = ?", bookmarks[i].Nftaddr).Find(&chat)

		//get num unread messages
		var chatCnt []entity.Groupchatitem
		var chatReadTime entity.Groupchatreadtime
		var dbQuery = database.Connector.Where("fromaddr = ?", key).Find(&chatReadTime)
		//if no respsonse to this query, its the first time a user is reading the chat history, send it all
		if dbQuery.RowsAffected == 0 {
			database.Connector.Where("nftaddr = ?", chat.Nftaddr).Find(&chatCnt)
		} else {
			database.Connector.Where("timestamp_dtm > ?", chatReadTime.Readtimestamp_dtm).Where("nftaddr = ?", chat.Nftaddr).Find(&chatCnt)
		}
		//end get num unread messages

		var returnItem entity.BookmarkReturnItem
		returnItem.Id = chat.Id
		returnItem.Lastmsg = chat.Message
		returnItem.Lasttimestamp = chat.Timestamp
		returnItem.Lasttimestamp_dtm = chat.Timestamp_dtm
		returnItem.Nftaddr = bookmarks[i].Nftaddr
		returnItem.Walletaddr = bookmarks[i].Walletaddr
		returnItem.Unreadcnt = len(chatCnt)
		if returnItem.Lastmsg == "" {
			var unsetTimeDtm time.Time
			var unsetTime string
			var noInt int
			returnItem.Unreadcnt = noInt
			returnItem.Lasttimestamp = unsetTime
			returnItem.Lasttimestamp_dtm = unsetTimeDtm
		}
		returnItems = append(returnItems, returnItem)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(returnItems)
}

// CreateImageItem godoc
// @Summary Store Image in DB for later user
// @Description Currently used for the WC HQ Logo, stores the base64 raw data of the profile image for a community
// @Tags Common
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param message body entity.Imageitem true "Profile Thumbnail Pic"
// @Success 200 {array} entity.Bookmarkitem
// @Router /v1/image [post]
func CreateImageItem(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var imgname entity.Imageitem
	json.Unmarshal(requestBody, &imgname)

	//Authuser := auth.GetUserFromReqContext(r)
	//TODO: get address to name mapping or send it in

	//if Authuser.Address == imgname. {
	database.Connector.Create(&imgname)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(imgname)
	// } else {
	// 	w.WriteHeader(http.StatusForbidden)
	// }
}

// UpdateImageItem godoc
// @Summary Store Image in DB for later user (update existing photo)
// @Description Currently used for the WC HQ Logo, stores the base64 raw data of the profile image for a community
// @Tags Common
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param message body entity.Imageitem true "Profile Thumbnail Pic"
// @Success 200 {array} entity.Bookmarkitem
// @Router /v1/image [put]
func UpdateImageItem(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var imgname entity.Imageitem
	json.Unmarshal(requestBody, &imgname)

	//TODO same as above, need address

	var result = database.Connector.Model(&entity.Addrnameitem{}).
		Where("name = ?", imgname.Name).
		Update("base64data", imgname.Base64data)

	var returnval bool
	if result.RowsAffected > 0 {
		returnval = true
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(returnval)
}

// GetImageItem godoc
// @Summary Get Thumbnail Image Data
// @Description Retreive image data for use with user/community/nft group dislayed icon
// @Tags Common
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param name path string true "Common Name Mapped to User/Community"
// @Success 200 {array} entity.Imageitem
// @Router /v1/image/{name} [get]
func GetImageItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	//TODO: need to confirm address is JWT for user

	var imgname []entity.Imageitem

	database.Connector.Where("name = ?", name).Find(&imgname)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(imgname)
}

// CreateAddrNameItem godoc
// @Summary give a common name to a user address, or NFT collection
// @Description Give a common name (Kevin.eth, BillyTheKid, etc) to an Address
// @Tags Common
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param message body entity.Addrnameitem true "Address and Name to map together"
// @Success 200 {array} entity.Bookmarkitem
// @Router /v1/name [post]
func CreateAddrNameItem(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var addrname entity.Addrnameitem
	json.Unmarshal(requestBody, &addrname)

	//Authuser := auth.GetUserFromReqContext(r)
	//if Authuser.Address == addrname.Address {
	database.Connector.Create(&addrname)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(addrname)
	// } else {
	// 	w.WriteHeader(http.StatusForbidden)
	// }
}

// GetAddrNameItem godoc
// @Summary get the common name which has been mapped to an address
// @Description get the given a common name (Kevin.eth, BillyTheKid, etc) what has already been mapped to an Address
// @Tags Common
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param address path string true "Get Name for given address"
// @Success 200 {array} entity.Addrnameitem
// @Router /v1/name/{name} [get]
func GetAddrNameItem(w http.ResponseWriter, r *http.Request) {
	//Authuser := auth.GetUserFromReqContext(r)
	//address := Authuser.Address
	vars := mux.Vars(r)
	address := vars["address"]

	var addrname []entity.Addrnameitem

	database.Connector.Where("address = ?", address).Find(&addrname)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(addrname)
}

// CreateAddrNameItem godoc
// @Summary give a common name to a user address, or NFT collection (update exiting)
// @Description Give a common name (Kevin.eth, BillyTheKid, etc) to an Address
// @Tags Common
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param message body entity.Addrnameitem true "Address and Name to map together"
// @Success 200 {array} entity.Bookmarkitem
// @Router /v1/name [put]
func UpdateAddrNameItem(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var addrname entity.Addrnameitem
	json.Unmarshal(requestBody, &addrname)

	Authuser := auth.GetUserFromReqContext(r)
	if Authuser.Address == addrname.Address {
		var result = database.Connector.Model(&entity.Addrnameitem{}).
			Where("address = ?", addrname.Address).
			Update("name", addrname.Name)

		var returnval bool
		if result.RowsAffected > 0 {
			returnval = true
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(returnval)
	} else {
		w.WriteHeader(http.StatusForbidden)
	}
}

// GetGroupChatItemsByAddr godoc
// @Summary Get group chat items, given a wallt FROM address and NFT Contract Address
// @Description Get all group chat items for a given wallet (useraddress) for a given NFT Contract Address (TODO: fix up var names)
// @Tags NFT
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param address path string true "NFT Address"
// @Param useraddress path string true "FROM: wallet address"
// @Success 200 {array} entity.Groupchatitem
// @Router /v1/get_groupchatitems/{address}/{useraddress} [get]
func GetGroupChatItemsByAddr(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nftaddr := vars["address"]
	//fromaddr := vars["useraddress"]
	Authuser := auth.GetUserFromReqContext(r)
	fromaddr := Authuser.Address

	var chat []entity.Groupchatitem

	var chatReadTime entity.Groupchatreadtime
	var dbQuery = database.Connector.Where("fromaddr = ?", fromaddr).Where("nftaddr = ?", nftaddr).Find(&chatReadTime)

	//fmt.Printf("Group Chat Get By Addr Result: %#v\n", chatReadTime)

	//if no respsonse to this query, its the first time a user is reading the chat history, send it all
	if dbQuery.RowsAffected == 0 {
		//database.Connector.Where("nftaddr = ?", nftaddr).Find(&chat)  //mana requests all data for now

		//add the first read element to the group timestamp table cross reference
		chatReadTime.Fromaddr = fromaddr
		chatReadTime.Nftaddr = nftaddr
		chatReadTime.Readtimestamp_dtm = time.Now()

		database.Connector.Create(&chatReadTime)
	} else {
		//database.Connector.Where("timestamp > ?", chatReadTime.Readtimestamp_dtm).Where("nftaddr = ?", nftaddr).Find(&chat) //mana requests all data for now
		//set timestamp when this was last grabbed
		currtime := time.Now()
		database.Connector.Model(&entity.Groupchatreadtime{}).Where("fromaddr = ?", fromaddr).Where("nftaddr = ?", nftaddr).Update("readtimestamp_dtm", currtime)
	}
	//this line goes away if we selectively load data in the future
	database.Connector.Where("nftaddr = ?", nftaddr).Find(&chat) //mana requests all data for now

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chat)
}

// GetGroupChatItemsByAddrLen godoc
// @Summary Get Unread Groupchat Items (TODO: cleanup naming convention here)
// @Description For group chat unread counts, currently the database stores a timestamp for each time a user enters a group chat.
// @Description We though in the design it would be impractical to keep a read/unread count copy per user per message, but if this
// @Description method doesn't proof to be fine grained enough, we could add a boolean relational table of read messgages per user.
// @Tags Common
// @Accept  json
// @Produce plain
// @Param name path string true "Common Name Mapped to User/Community"
// @Success 200 {integer} int
// @Router /v1/get_groupchatitems_unreadcnt/{address}/{useraddress} [get]
func GetGroupChatItemsByAddrLen(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nftaddr := vars["address"]
	//fromaddr := vars["useraddress"]
	Authuser := auth.GetUserFromReqContext(r)
	fromaddr := Authuser.Address

	var chat []entity.Groupchatitem

	var chatReadTime entity.Groupchatreadtime
	var dbQuery = database.Connector.Where("fromaddr = ?", fromaddr).Where("nftaddr = ?", nftaddr).Find(&chatReadTime)

	//fmt.Printf("Group Chat Get By Addr Result: %#v\n", chatReadTime.Readtimestamp_dtm)

	//if no respsonse to this query, its the first time a user is reading the chat history, send it all
	if dbQuery.RowsAffected == 0 {
		database.Connector.Where("nftaddr = ?", nftaddr).Find(&chat)
	} else {
		database.Connector.Where("timestamp_dtm > ?", chatReadTime.Readtimestamp_dtm).Where("nftaddr = ?", nftaddr).Find(&chat)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(len(chat))
}

// CreateAddrNameItem godoc
// @Summary Update Message Read Status of a given DM chat message
// @Description Currently this only update the message read/unread status.  It could update the entire JSON struct
// @Description upon request, however we only needed this functionality currently and it saved re-encryption of the data.
// @Description TODO: TO/FROM address in the URL is not needed/not used anymore.
// @Tags DMs
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param message body entity.Chatitem true "chat item JSON struct to update msg read status"
// @Success 200 {array} entity.Chatitem
// @Router /v1/update_chatitem/{fromaddr}/{toaddr} [put]
func UpdateChatitemByOwner(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var chat entity.Chatitem
	chat.Toaddr = strings.ToLower(chat.Toaddr)

	json.Unmarshal(requestBody, &chat)
	Authuser := auth.GetUserFromReqContext(r)
	if Authuser.Address == chat.Toaddr {
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
	} else {
		w.WriteHeader(http.StatusForbidden)
	}
}

// DeleteAllChatitemsToAddressByOwner godoc
// @Summary Delete All Chat Items (DMs) between FROM and TO given addresses
// @Description TODO: Need to protect this with JWT in addition to other API calls needed to use FROM addr from the JWT
// @Tags Unused/Legacy
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param toaddr path string true "TO: Address"
// @Param fromaddr path string true "FROM: Address"
// @Success 204
// @Router /v1/deleteall_chatitems/{fromaddr}/{toaddr} [delete]
func DeleteAllChatitemsToAddressByOwner(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	to := vars["toaddr"]
	//owner := vars["fromaddr"]

	var chat entity.Chatitem
	Authuser := auth.GetUserFromReqContext(r)
	owner := Authuser.Address

	database.Connector.Where("toaddr = ?", to).Where("fromaddr = ?", owner).Delete(&chat)
	w.WriteHeader(http.StatusNoContent)
}

// func CreateSettings(w http.ResponseWriter, r *http.Request) {
// 	requestBody, _ := ioutil.ReadAll(r.Body)
// 	var settings entity.Settings
// 	json.Unmarshal(requestBody, &settings)

// 	Authuser := auth.GetUserFromReqContext(r)
// 	if Authuser.Address == settings.Walletaddr {
// 		database.Connector.Create(&settings)
// 		w.Header().Set("Content-Type", "application/json")
// 		w.WriteHeader(http.StatusCreated)
// 		json.NewEncoder(w).Encode(settings)
// 	} else {
// 		w.WriteHeader(http.StatusForbidden)
// 	}
// }

// UpdateSettings godoc
// @Summary Settings hold a user address and the email address for notifications if they opt-in
// @Description Currently this only updates email address field
// @Tags Common
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param message body entity.Settings true "update struct"
// @Success 200 {array} entity.Settings
// @Router /v1/update_settings [POST]
func UpdateSettings(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var settings entity.Settings
	json.Unmarshal(requestBody, &settings)
	settings.Walletaddr = strings.ToLower(settings.Walletaddr)

	Authuser := auth.GetUserFromReqContext(r)
	if Authuser.Address == settings.Walletaddr {
		var dbResults = database.Connector.Model(&entity.Settings{}).Where("walletaddr = ?", settings.Walletaddr).Update("email", settings.Email)

		if dbResults.RowsAffected == 0 {
			database.Connector.Create(&settings)
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)
	} else {
		fmt.Println("UpdateSettings - JWT Address: ", Authuser.Address)
		fmt.Println("UpdateSettings - POST Address: ", settings.Walletaddr)
		w.WriteHeader(http.StatusForbidden)
	}
}

// DeleteSettings godoc
// @Summary Delete Settings Info
// @Description TODO: Need to protect this with JWT in addition to other API calls needed to use FROM addr from the JWT
// @Tags Unused/Legacy
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param address path string true "Wallet Address"
// @Success 204
// @Router /v1/delete_settings/{address} [delete]
func DeleteSettings(w http.ResponseWriter, r *http.Request) {
	//vars := mux.Vars(r)
	//key := vars["address"]
	Authuser := auth.GetUserFromReqContext(r)
	key := Authuser.Address

	var settings entity.Settings

	database.Connector.Where("walletaddr = ?", key).Delete(&settings)
	w.WriteHeader(http.StatusNoContent)
}

// GetSettings godoc
// @Summary Get Settings Info
// @Description TODO: Need to protect this with JWT in addition to other API calls needed to use FROM addr from the JWT
// @Tags Unused/Legacy
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param address path string true "Wallet Address"
// @Success 200 {array} entity.Settings
// @Router /v1/get_settings/{address} [get]
func GetSettings(w http.ResponseWriter, r *http.Request) {
	// vars := mux.Vars(r)
	// key := vars["address"]
	Authuser := auth.GetUserFromReqContext(r)
	key := Authuser.Address

	var settings []entity.Settings
	database.Connector.Where("walletaddr = ?", key).Find(&settings)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

// CreateComments godoc
// @Summary Comments are used within an NFT community chat
// @Description Comments are meant to be public, someday having an up/downvote method for auto-moderation
// @Tags NFT
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param message body entity.Comments true "create struct"
// @Success 200 {array} entity.Comments
// @Router /v1/create_comments [post]
func CreateComments(w http.ResponseWriter, r *http.Request) {
	requestBody, _ := ioutil.ReadAll(r.Body)
	var comment entity.Comments
	json.Unmarshal(requestBody, &comment)

	Authuser := auth.GetUserFromReqContext(r)
	if Authuser.Address == comment.Fromaddr {
		database.Connector.Create(&comment)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(comment)
	} else {
		w.WriteHeader(http.StatusForbidden)
	}
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

// DeleteComments godoc
// @Summary Delete Public Comments for given FROM wallet address, NFT Contract and ID
// @Description NFTs have a public comment section
// @Tags NFT
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param address path string true "FROM Wallet Address"
// @Param nftaddr path string true "NFT Contract Address"
// @Param nftid path string true "NFT ID"
// @Success 204
// @Router /v1/delete_comments/{fromaddr}/{nftaddr}/{nftid} [delete]
func DeleteComments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	//fromaddr := vars["address"]
	nftaddr := vars["nftaddr"]
	nftid := vars["nftid"]
	Authuser := auth.GetUserFromReqContext(r)
	fromaddr := Authuser.Address

	var comment entity.Comments

	database.Connector.Where("fromaddr = ?", fromaddr).Where("nftaddr = ?", nftaddr).Where("nftid = ?", nftid).Delete(&comment)
	w.WriteHeader(http.StatusNoContent)
}

// GetComments godoc
// @Summary Get Public Comments for given NFT Contract and ID
// @Description NFTs have a public comment section
// @Tags NFT
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param address path string true "Wallet Address"
// @Success 200 {array} entity.Comments
// @Router /v1/get_comments/{nftaddr}/{nftid} [get]
func GetComments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["nftid"]
	addr := vars["nftaddr"]

	var comment []entity.Comments
	database.Connector.Where("nftaddr = ?", addr).Where("nftid = ?", id).Find(&comment)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comment)
}

// GetCommentsCount godoc
// @Summary Get Public Comments Count for given NFT Contract and ID
// @Description NFTs have a public comment section
// @Tags NFT
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param nftaddr path string true "NFT Contract Address"
// @Param nftid path string true "NFT ID"
// @Success 200 {integer} int
// @Router /v1/get_comments_cnt/{nftaddr}/{nftid} [get]
func GetCommentsCount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["nftid"]
	addr := vars["nftaddr"]

	var comment []entity.Comments
	database.Connector.Where("nftaddr = ?", addr).Where("nftid = ?", id).Find(&comment)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(len(comment))
}

// func GetAllComments(w http.ResponseWriter, r *http.Request) {
// 	var comment []entity.Comments
// 	database.Connector.Find(&comment)

// 	//make sure to get the name if it wasn't there (not there by default now)
// 	var addrname entity.Addrnameitem
// 	for i := 0; i < len(comment); i++ {
// 		var result = database.Connector.Where("address = ?", comment[i].Fromaddr).Find(&addrname)
// 		if result.RowsAffected > 0 {
// 			comment[i].Name = addrname.Name
// 		}
// 	}
// 	//end of adding names for fromaddr

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(comment)
// }

func GetTwitter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	contract := vars["contract"]

	//slug := GetOpeseaSlug(contract)
	handle := GetTwitterHandle(contract)
	twitterID := GetTwitterID(handle)
	tweets := GetTweetsFromAPI(twitterID)
	formatted := FormatTwitterData(tweets)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(formatted)
}

func GetTwitterCount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	contract := vars["contract"]

	//slug := GetOpeseaSlug(contract)
	handle := GetTwitterHandle(contract)
	twitterID := GetTwitterID(handle)
	tweets := GetTweetsFromAPI(twitterID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(len(tweets.Data))
}

func GetTwitterHandle(contractAddr string) string {
	url := "https://api.opensea.io/api/v1/asset_contract/" + contractAddr

	// Create a new request using http
	req, _ := http.NewRequest("GET", url, nil)
	osKey := os.Getenv("OPENSEA_API_KEY")
	req.Header.Add("X-API-KEY", osKey)

	// Send req using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error while reading the response bytes:", err)
	}

	var result OpenseaData
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to the go struct pointer
		fmt.Println("Can not unmarshal JSON")
	}

	collection := result.Collection.TwitterUsername

	fmt.Printf("get twitter username: %#v\n", collection)

	return collection
}

func GetTwitterID(twitterHandle string) string {
	url := "https://api.twitter.com/2/users/by/username/" + twitterHandle

	// Create a Bearer string by appending string access token
	bearer := "Bearer " + os.Getenv("TWITTER_BEARER")

	// Create a new request using http
	req, _ := http.NewRequest("GET", url, nil)

	// add authorization header to the req
	req.Header.Add("Authorization", bearer)

	// Send req using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error while reading the response bytes:", err)
	}

	var result TwitterIdResp
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to the go struct pointer
		fmt.Println("Can not unmarshal JSON")
	}

	twitterID := result.Data.Id

	//fmt.Printf("get twitter ID: %#v\n", twitterID)

	return twitterID
}

func GetTweetsFromAPI(twitterID string) TwitterTweetsData {
	//url := "https://api.twitter.com/2/users/" + twitterID + "/tweets"
	url := "https://api.twitter.com/2/users/" + twitterID + "/tweets?media.fields=height,width,url,preview_image_url,type&tweet.fields=attachments,created_at&user.fields=profile_image_url,username&expansions=author_id,attachments.media_keys&exclude=retweets"

	// Create a Bearer string by appending string access token
	bearer := "Bearer " + os.Getenv("TWITTER_BEARER")

	// Create a new request using http
	req, _ := http.NewRequest("GET", url, nil)

	// add authorization header to the req
	req.Header.Add("Authorization", bearer)

	// Send req using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error fetching twitter: ", err)
		log.Println("Error on response.\n[ERROR] -", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error fetching twitter bytes: ", err)
		log.Println("Error while reading the response bytes:", err)
	}

	//fmt.Println("body twitter: ", body)

	var result TwitterTweetsData
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to the go struct pointer
		fmt.Println("Can not unmarshal JSON")
	}
	//fmt.Println("length twitter: ", len(result.Data))

	return result
}

func FormatTwitterData(data TwitterTweetsData) []TweetType {
	var tweets []TweetType
	if len(data.Data) > 0 {
		var user User
		if len(data.Includes.Users) > 0 {
			user = data.Includes.Users[0]
		}

		//for i, item := range data.data {
		//first copy just data.data stuff
		for i := 0; i < len(data.Data); i++ {
			// Text        string `json:"text"`
			// ID          string `json:"id"`
			// Attachments struct {
			// 	MediaKeys []string `json:"media_keys"`
			// } `json:"attachments"`
			// AuthorID  string    `json:"author_id"`
			// CreatedAt time.Time `json:"created_at"`
			var initData TweetType
			initData.Text = data.Data[i].Text
			initData.ID = data.Data[i].ID
			initData.Attachments = data.Data[i].Attachments
			initData.AuthorID = data.Data[i].AuthorID
			initData.CreatedAt = data.Data[i].CreatedAt
			tweets = append(tweets, initData)
		}

		for i := 0; i < len(data.Data); i++ {
			tweets[i].User = user

			if len(data.Data[i].Attachments.MediaKeys) > 0 {
				var localAttachment Attachments
				for j := 0; j < len(tweets[i].Attachments.MediaKeys); j++ {
					var mediaKey = tweets[i].Attachments.MediaKeys[j]
					if len(data.Includes.Media) > 0 {
						//var matched = data.includes.media.find((item => item.media_key === mediaKey))
						for _, v := range data.Includes.Media {
							if v.MediaKey == mediaKey {
								if v.URL != "" {
									localAttachment.MediaKeys = append(localAttachment.MediaKeys, v.URL)
								}
							}
						}
					}
				}
				if len(localAttachment.MediaKeys) > 0 {
					tweets[i].Media = localAttachment
				}
			}
		}
	}
	return tweets
}

// GetCommunityChat godoc
// @Summary Get Community Chat Landing Page Info
// @Description TODO: need a creation API for communities, which includes specificied welcome message text, Twitter handle, page title
// @Tags GroupChat
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param address path string true "Wallet Address"
// @Param address path string true "Wallet Address"
// @Success 200 {array} LandingPageItems
// @Router /v1/community/{community}/{address} [get]
func GetCommunityChat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	community := vars["community"]
	key := vars["address"]
	var landingData LandingPageItems

	//todo someday this will be for general communities (need to rename functions)
	//fmt.Printf("Get WalletChat HQ: %#v\n", community)

	//for now, the walletchat living room is all users by default
	var members []entity.Bookmarkitem
	database.Connector.Where("nftaddr = ?", community).Find(&members)
	landingData.Members = len(members)

	//name
	landingData.Name = "WalletChat HQ" //TODO this should come from a table which stores info set in in a CREATE community chat table

	//logo base64 data (url requires other changes)
	var imgname entity.Imageitem
	database.Connector.Where("name = ?", community).Find(&imgname)
	landingData.Logo = imgname.Base64data

	//WalletChat is verified of course
	landingData.Verified = true

	//auto-join new users to WalletChat community (they can leave later)
	var bookmarks []entity.Bookmarkitem
	var dbQuery = database.Connector.Where("nftaddr = ?", community).Where("walletaddr = ?", key).Find(&bookmarks)
	if dbQuery.RowsAffected == 0 {
		var bookmark entity.Bookmarkitem
		bookmark.Nftaddr = community
		bookmark.Walletaddr = key
		bookmark.Chain = "none"

		database.Connector.Create(&bookmark)

		//by default everyone is joined to Walletchat
		landingData.Joined = true
		//create the welcome message, save it
		var newgroupchatuser entity.Groupchatitem
		newgroupchatuser.Type = entity.Welcome
		newgroupchatuser.Contexttype = entity.Community
		newgroupchatuser.Fromaddr = key
		newgroupchatuser.Nftaddr = community
		newgroupchatuser.Message = "Welcome " + key + " to Wallet Chat HQ!" //TODO end of this message should come from a table which stores info set in in a CREATE community chat table
		newgroupchatuser.Timestamp_dtm = time.Now()
		newgroupchatuser.Timestamp = time.Now().Format("2006-01-02T15:04:05.000Z")

		//add it to the database
		database.Connector.Create(&newgroupchatuser)
	} else {
		//We don't have a way for users to get back to WC HQ if they leave (shouldn't need to use above block to re-welcome them)
		landingData.Joined = true
	}

	//check messages read for this user address because this GetCommunityChat is being called
	//separately each time (I thought it would be filled from bookmarks)
	var groupchat []entity.Groupchatitem
	database.Connector.Where("nftaddr = ?", community).Where("fromaddr = ?", key).Find(&groupchat)
	//redoing some things already done in getGroupChatItemsByAddr
	var chatReadTime entity.Groupchatreadtime
	dbQuery = database.Connector.Where("fromaddr = ?", key).Where("nftaddr = ?", community).Find(&chatReadTime)
	if dbQuery.RowsAffected == 0 {
		//add the first read element to the group timestamp table cross reference
		chatReadTime.Fromaddr = key
		chatReadTime.Nftaddr = community
		chatReadTime.Readtimestamp_dtm = time.Now()

		database.Connector.Create(&chatReadTime)
	} else {
		//database.Connector.Where("timestamp > ?", chatReadTime.Readtimestamp_dtm).Where("nftaddr = ?", nftaddr).Find(&chat) //mana requests all data for now
		//set timestamp when this was last grabbed
		currtime := time.Now()
		database.Connector.Model(&entity.Groupchatreadtime{}).Where("fromaddr = ?", key).Where("nftaddr = ?", community).Update("readtimestamp_dtm", currtime)
	}

	var hasMessaged bool
	if len(groupchat) > 0 {
		hasMessaged = true
	} else {
		hasMessaged = false
	}
	landingData.Messaged = hasMessaged

	//grab all the data for walletchat group
	database.Connector.Where("nftaddr = ?", community).Find(&groupchat)
	landingData.Messages = groupchat

	//get twitter data
	twitterID := GetTwitterID("wallet_chat") //could get this once and hardcode the ID in here too to save one API call
	tweets := GetTweetsFromAPI(twitterID)
	formatted := FormatTwitterData(tweets)
	landingData.Tweets = formatted

	//social data
	var twitterSocial SocialMsg
	twitterSocial.Type = "twitter"
	twitterSocial.Username = "@wallet_chat" //TODO this should come from a table which stores info set in in a CREATE community chat table
	landingData.Social = append(landingData.Social, twitterSocial)
	var discordSocial SocialMsg
	discordSocial.Type = "discord"
	discordSocial.Username = "WalletChat" //TODO this should come from a table which stores info set in in a CREATE community chat table
	landingData.Social = append(landingData.Social, discordSocial)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(landingData)
}

// IsOwner godoc
// @Summary Check if given wallet address owns an NFT from given contract address
// @Description API user could check this directly via any third party service like NFTPort, Moralis as well
// @Tags Common
// @Accept  json
// @Produce  json
// @Security BearerAuth
// @Param contract path string true "NFT Contract Address"
// @Param wallet path string true "Wallet Address"
// @Success 200 {array} LandingPageItems
// @Router /v1/is_owner/{contract}/{wallet} [get]
func IsOwner(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	contract := vars["contract"]
	wallet := vars["wallet"]

	result := IsOwnerOfNFT(contract, wallet, "ethereum")
	if !result {
		result = IsOwnerOfNFT(contract, wallet, "polygon")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

//internal
func GetOwnerNFTs(walletAddr string, chain string) NFTPortOwnerOf {
	//url := "https://eth-mainnet.alchemyapi.io/v2/${process.env.REACT_APP_ALCHEMY_API_KEY}/getOwnersForToken" + contractAddr
	url := "https://api.nftport.xyz/v0/accounts/" + walletAddr + "?chain=" + chain

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", os.Getenv("NFTPORT_API_KEY"))

	// Send req using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error while reading the response bytes:", err)
	}

	var result NFTPortOwnerOf
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to the go struct pointer
		fmt.Println("Can not unmarshal JSON")
	}

	//fmt.Printf("IsOwner: %#v\n", result.Total)

	return result
}

//internal use
func IsOwnerOfNFT(contractAddr string, walletAddr string, chain string) bool {
	//url := "https://eth-mainnet.alchemyapi.io/v2/${process.env.REACT_APP_ALCHEMY_API_KEY}/getOwnersForToken" + contractAddr
	url := "https://api.nftport.xyz/v0/accounts/" + walletAddr + "?chain=" + chain + "&contract_address=" + contractAddr

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", os.Getenv("NFTPORT_API_KEY"))

	// Send req using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error while reading the response bytes:", err)
	}

	var result NFTPortOwnerOf
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to the go struct pointer
		fmt.Println("Can not unmarshal JSON")
	}

	//fmt.Printf("IsOwner: %#v\n", result.Total)

	return result.Total > 0
}

func IsOnChain(contractAddr string, chain string) bool {
	url := "https://api.nftport.xyz/v0/nfts/" + contractAddr + "?chain=" + chain

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", os.Getenv("NFTPORT_API_KEY"))

	// Send req using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error while reading the response bytes:", err)
	}

	var result NFTPortNftContract
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to the go struct pointer
		fmt.Println("Can not unmarshal JSON")
	}

	//fmt.Printf("Chain Response: %#v\n", result.Response)

	var returnVal = false
	if result.Response == "OK" {
		returnVal = true
	}
	return returnVal
}

//this was just used to fix up users info after adding new column
//not intended for extenal calls
func FixUpBookmarks(w http.ResponseWriter, r *http.Request) {
	var bookmarks []entity.Bookmarkitem
	database.Connector.Find(&bookmarks)

	for _, bookmark := range bookmarks {
		if strings.HasPrefix(bookmark.Nftaddr, "0x") {
			var result = IsOnChain(bookmark.Nftaddr, "ethereum")
			if result {
				database.Connector.Model(&entity.Bookmarkitem{}).Where("walletaddr = ?", bookmark.Walletaddr).Where("nftaddr = ?", bookmark.Nftaddr).Update("chain", "ethereum")
			} else {
				var result = IsOnChain(bookmark.Nftaddr, "polygon")
				if result {
					database.Connector.Model(&entity.Bookmarkitem{}).Where("walletaddr = ?", bookmark.Walletaddr).Where("nftaddr = ?", bookmark.Nftaddr).Update("chain", "polygon")
				}
			}
		}
	}
}

//internal use only
func AutoJoinCommunities(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	walletAddr := vars["wallet"]
	AutoJoinCommunitiesByChain(walletAddr, "ethereum")
	AutoJoinCommunitiesByChain(walletAddr, "polygon")
	AutoJoinPoapChats(walletAddr)
}

//internal use only
func AutoJoinCommunitiesByChain(walletAddr string, chain string) {
	//TODO: OS is more accurate
	url := "https://api.nftport.xyz/v0/accounts/" + walletAddr + "?chain=" + chain

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", os.Getenv("NFTPORT_API_KEY"))

	// Send req using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error while reading the response bytes:", err)
	}

	var result NFTPortOwnerOf
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to the go struct pointer
		fmt.Println("Can not unmarshal JSON")
	}

	//fmt.Printf("IsOwner: %#v\n", result.Total)
	for _, nft := range result.Nfts {
		//TODO: could be optimized, good enough for now
		var bookmarkExists entity.Bookmarkitem
		var dbResult = database.Connector.Where("nftaddr = ?", nft.ContractAddress).Where("walletaddr = ?", walletAddr).Find(&bookmarkExists)

		if dbResult.RowsAffected == 0 {
			//check if the user already manually unjoined, if so don't auto rejoin them
			var userUnjoined entity.Userunjoined
			var dbUnjoined = database.Connector.Where("nftaddr = ?", nft.ContractAddress).Where("walletaddr = ?", walletAddr).Find(&userUnjoined)
			userAlreadyUnjoined := false
			if dbUnjoined.RowsAffected > 0 {
				userAlreadyUnjoined = userUnjoined.Unjoined
			}

			if !userAlreadyUnjoined {
				fmt.Println("Found new NFT: " + nft.ContractAddress)
				var bookmark entity.Bookmarkitem

				bookmark.Nftaddr = nft.ContractAddress
				bookmark.Walletaddr = walletAddr
				bookmark.Chain = chain

				database.Connector.Create(&bookmark)
			}
		}
	}
}

//internal use only
func AutoJoinPoapChats(walletAddr string) {
	//https://documentation.poap.tech/reference/getactionsscan-5
	var poapInfo []POAPInfoByAddress = getPoapInfoByAddress(walletAddr)
	//fmt.Printf("AutoJoinPoapChats: %#v\n", poapInfo)
	for _, poap := range poapInfo {
		var bookmarkExists entity.Bookmarkitem

		var poapAddr = "poap_" + strconv.Itoa(poap.Event.ID)
		//fmt.Printf("POAP Event: %#v\n", poapAddr)
		var dbResult = database.Connector.Where("nftaddr = ?", poapAddr).Where("walletaddr = ?", walletAddr).Find(&bookmarkExists)

		if dbResult.RowsAffected == 0 {
			//check if the user already manually unjoined, if so don't auto rejoin them
			var userUnjoined entity.Userunjoined
			var dbUnjoined = database.Connector.Where("nftaddr = ?", poapAddr).Where("walletaddr = ?", walletAddr).Find(&userUnjoined)
			userAlreadyUnjoined := false
			if dbUnjoined.RowsAffected > 0 {
				userAlreadyUnjoined = userUnjoined.Unjoined
			}

			if !userAlreadyUnjoined {
				fmt.Printf("POAP is new for user: %#v\n", walletAddr)
				var bookmark entity.Bookmarkitem

				bookmark.Nftaddr = poapAddr
				bookmark.Walletaddr = walletAddr
				bookmark.Chain = poap.Chain

				database.Connector.Create(&bookmark)
			}
		}
	}
}

func GetPoapsByAddr(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	walletAddr := vars["wallet"]

	result := getPoapInfoByAddress(walletAddr)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

//internal use only
func getPoapInfoByAddress(walletAddr string) []POAPInfoByAddress {
	var result []POAPInfoByAddress
	url := "https://api.poap.tech/actions/scan/" + walletAddr

	// Create a new request using http
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("X-API-KEY", os.Getenv("POAP_API_KEY"))

	// Send req using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	} else {
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("Error while reading the response bytes:", err)
		}

		if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to the go struct pointer
			fmt.Println("Can not unmarshal JSON")
		}
	}

	//fmt.Printf("returning: %#v\n", result)

	return result
}

type POAPInfoByAddress struct {
	Event struct {
		ID          int    `json:"id"`
		FancyID     string `json:"fancy_id"`
		Name        string `json:"name"`
		EventURL    string `json:"event_url"`
		ImageURL    string `json:"image_url"`
		Country     string `json:"country"`
		City        string `json:"city"`
		Description string `json:"description"`
		Year        int    `json:"year"`
		StartDate   string `json:"start_date"`
		EndDate     string `json:"end_date"`
		ExpiryDate  string `json:"expiry_date"`
		Supply      int    `json:"supply"`
	} `json:"event"`
	TokenID string `json:"tokenId"`
	Owner   string `json:"owner"`
	Chain   string `json:"chain"`
	Created string `json:"created"`
}

type NFTPortNftContract struct {
	Response string `json:"response"`
	Nfts     []struct {
		Chain           string `json:"chain"`
		ContractAddress string `json:"contract_address"`
		TokenID         string `json:"token_id"`
	} `json:"nfts"`
	Contract struct {
		Name     string `json:"name"`
		Symbol   string `json:"symbol"`
		Type     string `json:"type"`
		Metadata struct {
			Description        string `json:"description"`
			ThumbnailURL       string `json:"thumbnail_url"`
			CachedThumbnailURL string `json:"cached_thumbnail_url"`
			BannerURL          string `json:"banner_url"`
			CachedBannerURL    string `json:"cached_banner_url"`
		} `json:"metadata"`
	} `json:"contract"`
	Total int `json:"total"`
}

type User struct {
	Username        string `json:"username"`
	ProfileImageURL string `json:"profile_image_url"`
	ID              string `json:"id"`
	Name            string `json:"name"`
}

type Attachments struct {
	MediaKeys []string `json:"media_keys"`
}

type TwitterTweetsData struct {
	Data []struct {
		Text        string `json:"text"`
		ID          string `json:"id"`
		Attachments struct {
			MediaKeys []string `json:"media_keys"`
		} `json:"attachments,omitempty"`
		AuthorID  string    `json:"author_id"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"data"`
	Includes struct {
		Media []struct {
			Type            string `json:"type"`
			Width           int    `json:"width"`
			PreviewImageURL string `json:"preview_image_url,omitempty"`
			Height          int    `json:"height"`
			MediaKey        string `json:"media_key"`
			URL             string `json:"url,omitempty"`
		} `json:"media"`
		Users []struct {
			Username        string `json:"username"`
			ProfileImageURL string `json:"profile_image_url"`
			ID              string `json:"id"`
			Name            string `json:"name"`
		} `json:"users"`
	} `json:"includes"`
	Meta struct {
		NextToken   string `json:"next_token"`
		ResultCount int    `json:"result_count"`
		NewestID    string `json:"newest_id"`
		OldestID    string `json:"oldest_id"`
	} `json:"meta"`
}

//formatted for use in client side per Mana
type TweetType struct {
	Text        string `json:"text"`
	ID          string `json:"id"`
	Attachments struct {
		MediaKeys []string `json:"media_keys"`
	} `json:"attachments"`
	AuthorID  string    `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
	User      struct {
		Username        string `json:"username"`
		ProfileImageURL string `json:"profile_image_url"`
		ID              string `json:"id"`
		Name            string `json:"name"`
	} `json:"user"`
	Media Attachments `json:"media"`
}

type TwitterIdResp struct {
	Data struct {
		Id       string `json:"id"`
		Name     string `json:"name"`
		Username string `json:"username"`
	} `json:"data"`
}

type Social struct {
	SocialMsg []string `json:"social"`
}

type SocialMsg struct {
	Type     string `json:"type"`
	Username string `json:"username"`
}

type LandingPageItems struct {
	Name     string                 `json:"name"`
	Members  int                    `json:"members"`
	Logo     string                 `json:"logo"`         // logo url, stored in backend
	Verified bool                   `json:"is_verified"`  // is this group verified? WalletChat's group is verified by default
	Joined   bool                   `json:"joined"`       //number of members of the group
	Messaged bool                   `json:"has_messaged"` // has user messaged in this group chat before? if not show "Say hi" button
	Messages []entity.Groupchatitem `json:"messages"`
	Tweets   []TweetType            `json:"tweets"` // follow format of GET /get_twitter/{nftAddr}
	Social   []SocialMsg            `json:"social"`
}

type OpenseaData struct {
	Collection struct {
		BannerImageURL          string      `json:"banner_image_url"`
		ChatURL                 interface{} `json:"chat_url"`
		CreatedDate             string      `json:"created_date"`
		DefaultToFiat           bool        `json:"default_to_fiat"`
		Description             string      `json:"description"`
		DevBuyerFeeBasisPoints  string      `json:"dev_buyer_fee_basis_points"`
		DevSellerFeeBasisPoints string      `json:"dev_seller_fee_basis_points"`
		DiscordURL              string      `json:"discord_url"`
		DisplayData             struct {
			CardDisplayStyle string `json:"card_display_style"`
		} `json:"display_data"`
		ExternalURL                 string      `json:"external_url"`
		Featured                    bool        `json:"featured"`
		FeaturedImageURL            string      `json:"featured_image_url"`
		Hidden                      bool        `json:"hidden"`
		SafelistRequestStatus       string      `json:"safelist_request_status"`
		ImageURL                    string      `json:"image_url"`
		IsSubjectToWhitelist        bool        `json:"is_subject_to_whitelist"`
		LargeImageURL               string      `json:"large_image_url"`
		MediumUsername              string      `json:"medium_username"`
		Name                        string      `json:"name"`
		OnlyProxiedTransfers        bool        `json:"only_proxied_transfers"`
		OpenseaBuyerFeeBasisPoints  string      `json:"opensea_buyer_fee_basis_points"`
		OpenseaSellerFeeBasisPoints string      `json:"opensea_seller_fee_basis_points"`
		PayoutAddress               string      `json:"payout_address"`
		RequireEmail                bool        `json:"require_email"`
		ShortDescription            interface{} `json:"short_description"`
		Slug                        string      `json:"slug"`
		TelegramURL                 interface{} `json:"telegram_url"`
		TwitterUsername             string      `json:"twitter_username"`
		InstagramUsername           string      `json:"instagram_username"`
		WikiURL                     interface{} `json:"wiki_url"`
		IsNsfw                      bool        `json:"is_nsfw"`
	} `json:"collection"`
	Address                     string      `json:"address"`
	AssetContractType           string      `json:"asset_contract_type"`
	CreatedDate                 string      `json:"created_date"`
	Name                        string      `json:"name"`
	NftVersion                  string      `json:"nft_version"`
	OpenseaVersion              interface{} `json:"opensea_version"`
	Owner                       int         `json:"owner"`
	SchemaName                  string      `json:"schema_name"`
	Symbol                      string      `json:"symbol"`
	TotalSupply                 string      `json:"total_supply"`
	Description                 string      `json:"description"`
	ExternalLink                string      `json:"external_link"`
	ImageURL                    string      `json:"image_url"`
	DefaultToFiat               bool        `json:"default_to_fiat"`
	DevBuyerFeeBasisPoints      int         `json:"dev_buyer_fee_basis_points"`
	DevSellerFeeBasisPoints     int         `json:"dev_seller_fee_basis_points"`
	OnlyProxiedTransfers        bool        `json:"only_proxied_transfers"`
	OpenseaBuyerFeeBasisPoints  int         `json:"opensea_buyer_fee_basis_points"`
	OpenseaSellerFeeBasisPoints int         `json:"opensea_seller_fee_basis_points"`
	BuyerFeeBasisPoints         int         `json:"buyer_fee_basis_points"`
	SellerFeeBasisPoints        int         `json:"seller_fee_basis_points"`
	PayoutAddress               string      `json:"payout_address"`
}
