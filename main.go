package main

import (
	"log"
	"net/http"
	"rest-go-demo/controllers"
	"rest-go-demo/database"

	"github.com/gorilla/mux"
	_ "github.com/jinzhu/gorm/dialects/mysql" //Required for MySQL dialect
)

func main() {
	initDB()
	log.Println("Starting the HTTP server on port 8080")

	router := mux.NewRouter().StrictSlash(true)
	initaliseHandlers(router)
	log.Fatal(http.ListenAndServe(":8080", router))
}

func initaliseHandlers(router *mux.Router) {
	//1-to-1 chats (both general and NFT related)
	router.HandleFunc("/get_unread_cnt/{address}", controllers.GetUnreadMsgCntTotal).Methods("GET")
	router.HandleFunc("/get_unread_cnt/{fromaddr}/{toaddr}", controllers.GetUnreadMsgCnt).Methods("GET")
	router.HandleFunc("/get_unread_cnt/{address}/{nftaddr}/{nftid}", controllers.GetUnreadMsgCntNft).Methods("GET")
	router.HandleFunc("/get_unread_cnt_nft/{address}", controllers.GetUnreadMsgCntNftAllByAddr).Methods("GET")
	router.HandleFunc("/getall_chatitems/{address}", controllers.GetChatFromAddress).Methods("GET")
	router.HandleFunc("/getall_chatitems/{fromaddr}/{toaddr}", controllers.GetChatFromAddressToAddr).Methods("GET")
	router.HandleFunc("/getnft_chatitems/{fromaddr}/{toaddr}/{nftaddr}/{nftid}", controllers.GetChatNftAllItemsFromAddrAndNFT).Methods("GET")
	router.HandleFunc("/getnft_chatitems/{address}/{nftaddr}/{nftid}", controllers.GetChatNftAllItemsFromAddr).Methods("GET")
	router.HandleFunc("/getnft_chatitems/{nftaddr}/{nftid}", controllers.GetChatNftContext).Methods("GET")
	router.HandleFunc("/getnft_chatitems/{address}", controllers.GetNftChatFromAddress).Methods("GET")
	router.HandleFunc("/update_chatitem/{fromaddr}/{toaddr}", controllers.UpdateChatitemByOwner).Methods("PUT")
	router.HandleFunc("/deleteall_chatitems/{address}", controllers.DeleteAllChatitemsToAddressByOwner).Methods("DELETE")
	router.HandleFunc("/get_inbox/{address}", controllers.GetInboxByOwner).Methods("GET")
	router.HandleFunc("/create_chatitem", controllers.CreateChatitem).Methods("POST")
	//router.HandleFunc("/create_chatitem_tmp", controllers.CreateChatitemTmp).Methods("POST")
	router.HandleFunc("/getall_chatitems", controllers.GetAllChatitems).Methods("GET")

	//group chat
	router.HandleFunc("/create_groupchatitem", controllers.CreateGroupChatitem).Methods("POST")
	router.HandleFunc("/get_groupchatitems/{address}", controllers.GetGroupChatItems).Methods("GET")
	router.HandleFunc("/get_groupchatitems/{address}/{useraddress}", controllers.GetGroupChatItemsByAddr).Methods("GET")
	router.HandleFunc("/get_groupchatitems_unreadcnt/{address}/{useraddress}", controllers.GetGroupChatItemsByAddrLen).Methods("GET")

	//group chat
	router.HandleFunc("/community/{community}/{address}", controllers.GetWalletChat).Methods("GET") //TODO: make common
	router.HandleFunc("/community", controllers.CreateCommunityChatitem).Methods("POST")

	//bookmarks
	router.HandleFunc("/create_bookmark", controllers.CreateBookmarkItem).Methods("POST")
	router.HandleFunc("/delete_bookmark", controllers.DeleteBookmarkItem).Methods("POST")
	router.HandleFunc("/get_bookmarks/{address}", controllers.GetBookmarkItems).Methods("GET")
	router.HandleFunc("/get_bookmarks/{walletaddr}/{nftaddr}", controllers.IsBookmarkItem).Methods("GET")

	//naming addresses (users or NFT collections)
	router.HandleFunc("/name", controllers.CreateAddrNameItem).Methods("POST")
	router.HandleFunc("/name", controllers.UpdateAddrNameItem).Methods("PUT")
	router.HandleFunc("/name/{address}", controllers.GetAddrNameItem).Methods("GET")

	//Logos / Images stored in base64
	router.HandleFunc("/image", controllers.CreateImageItem).Methods("POST")
	router.HandleFunc("/image", controllers.UpdateImageItem).Methods("PUT")
	router.HandleFunc("/image/{name}", controllers.GetImageItem).Methods("GET")

	//settings items - currently this is the public key added upon first login for encryption/signing without MM
	router.HandleFunc("/create_settings", controllers.CreateSettings).Methods("POST")
	router.HandleFunc("/update_settings", controllers.UpdateSettings).Methods("PUT")
	router.HandleFunc("/get_settings/{address}", controllers.GetSettings).Methods("GET")
	router.HandleFunc("/delete_settings/{address}", controllers.DeleteSettings).Methods("DELETE")

	//comments on a specific NFT
	router.HandleFunc("/create_comments", controllers.CreateComments).Methods("POST")
	router.HandleFunc("/get_comments", controllers.GetAllComments).Methods("GET")
	router.HandleFunc("/get_comments/{nftaddr}/{nftid}", controllers.GetComments).Methods("GET")
	router.HandleFunc("/delete_comments/{fromaddr}/{nftaddr}/{nftid}", controllers.DeleteComments).Methods("DELETE")

	//Twitter Related APIs
	router.HandleFunc("/get_twitter/{contract}", controllers.GetTwitter).Methods("GET")
	router.HandleFunc("/get_twitter_cnt/{contract}", controllers.GetTwitterCount).Methods("GET")
	router.HandleFunc("/get_comments_cnt/{nftaddr}/{nftid}", controllers.GetCommentsCount).Methods("GET")
}

func initDB() {
	config :=
		database.Config{
			User:       "doadmin",
			Password:   "AVNS_7q8_Jqll_0sA9Fi",
			ServerName: "db-mysql-nyc3-11937-do-user-11094376-0.b.db.ondigitalocean.com:25060",
			DB:         "walletchat",
		}
		// database.Config{
		// 	User:       "root",
		// 	Password:   "",
		// 	ServerName: "localhost:3306",
		// 	DB:         "walletchat",
		// }

	connectionString := database.GetConnectionString(config)
	err := database.Connect(connectionString)
	if err != nil {
		panic(err.Error())
	}

	//These are supposed to help create the proper DB based on the data struct if it doesn't already exist
	//had some issues with it and just created the tables directly in MySQL (still have to match data structs)
	// database.Migrate(&entity.Settings{})
	//database.MigrateComments(&entity.Comments{})
	// database.MigrateChatitem(&entity.Chatitem{})
}
