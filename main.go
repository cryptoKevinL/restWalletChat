package main

import (
	"log"
	"net/http"
	"os"
	"rest-go-demo/controllers"
	"rest-go-demo/database"

	"github.com/joho/godotenv"

	_ "rest-go-demo/docs" // docs is generated by Swag CLI, you have to import it

	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/gorilla/mux"
	_ "github.com/jinzhu/gorm/dialects/mysql" //Required for MySQL dialect
)

// @title WalletChat API
// @version 0.0
// @description Wecome to the WalletChat API Documentation
// @description
// @description Please make note that some JSON data structures are shared for both input/output.
// @description Required input parameters will have a red * next to them in the data type outline at
// @description the bottom of the page, along with a comment.  This means when executing API functionality
// @description from this API page, some fields may need to be removed from the JSON struct before submitting.
// @description Please email the developers with any issues.
// @description Some JSON data structures are output only, and will be marked as such as well.
// @description
// @description v0 of the API does not include encryption or authentication.  Please as you are given access
// @description to this page, do not abuse this system and impersonate others, or submit offensive material.
// @description developers monitor this data daily.
// @description
// @description v1 will include basic JWT Authentication, however some additional work is in progress to make this fully secure.
// @description except for AUTH functions, all endpoints must prefix /v1 and include "Bearer: " in all reqests"

// @description v2 will include encyrption for DMs, private keys will be stored locally on client PCs
// @description with no way for us to recover any data which is encrypted.

// @wallet_chat API Support via Twitter
// @contact.url https://walletchat.fun
// @contact.email walletchatextension@gmail.com

// @host restwalletchat-app-sey3k.ondigitalocean.app
// @BasePath
func main() {
	godotenv.Load(".env")
	initDB()
	log.Println("Starting the HTTP server on port 8080")

	router := mux.NewRouter().StrictSlash(true)
	router.PathPrefix("/docs").Handler(httpSwagger.WrapHandler)
	initaliseHandlers(router)
	log.Fatal(http.ListenAndServe(":8080", router))
}

func initaliseHandlers(router *mux.Router) {
	//1-to-1 chats (both general and NFT related)
	router.HandleFunc("/get_unread_cnt/{address}", controllers.GetUnreadMsgCntTotal).Methods("GET")
	router.HandleFunc("/get_unread_cnt_by_type/{address}/{type}", controllers.GetUnreadMsgCntTotalByType).Methods("GET")
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
	router.HandleFunc("/deleteall_chatitems/{fromaddr}/{toaddr}", controllers.DeleteAllChatitemsToAddressByOwner).Methods("DELETE")
	router.HandleFunc("/get_inbox/{address}", controllers.GetInboxByOwner).Methods("GET")
	router.HandleFunc("/create_chatitem", controllers.CreateChatitem).Methods("POST")
	//router.HandleFunc("/create_chatitem_tmp", controllers.CreateChatitemTmp).Methods("POST")
	//router.HandleFunc("/getall_chatitems", controllers.GetAllChatitems).Methods("GET")

	//unreadcnt per week4 requirements
	router.HandleFunc("/unreadcount/{address}", controllers.GetUnreadcnt).Methods("GET")
	//router.HandleFunc("/unreadcount/{address}", controllers.PutUnreadcnt).Methods("PUT")

	//group chat
	router.HandleFunc("/create_groupchatitem", controllers.CreateGroupChatitem).Methods("POST")
	router.HandleFunc("/get_groupchatitems/{address}", controllers.GetGroupChatItems).Methods("GET")
	router.HandleFunc("/get_groupchatitems/{address}/{useraddress}", controllers.GetGroupChatItemsByAddr).Methods("GET")
	router.HandleFunc("/get_groupchatitems_unreadcnt/{address}/{useraddress}", controllers.GetGroupChatItemsByAddrLen).Methods("GET")

	//group chat
	//TODO: we need a create community API call, which provides twitter/discord handles, welcome message, Title/Name (see hardcoded items in GetCommunityChat)
	router.HandleFunc("/community/{community}/{address}", controllers.GetCommunityChat).Methods("GET") //TODO: make common
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
	//router.HandleFunc("/get_comments", controllers.GetAllComments).Methods("GET") //doubt we will need this
	router.HandleFunc("/get_comments/{nftaddr}/{nftid}", controllers.GetComments).Methods("GET")
	router.HandleFunc("/delete_comments/{fromaddr}/{nftaddr}/{nftid}", controllers.DeleteComments).Methods("DELETE")

	//Twitter Related APIs
	router.HandleFunc("/get_twitter/{contract}", controllers.GetTwitter).Methods("GET")
	router.HandleFunc("/get_twitter_cnt/{contract}", controllers.GetTwitterCount).Methods("GET")
	router.HandleFunc("/get_comments_cnt/{nftaddr}/{nftid}", controllers.GetCommentsCount).Methods("GET")

	//holder functions
	//TODO: this would need a signature from holder to fully verify - ok for now
	router.HandleFunc("/is_owner/{contract}/{wallet}", controllers.IsOwner).Methods("GET")
	router.HandleFunc("/rejoin_all/{wallet}", controllers.AutoJoinCommunities).Methods("GET")
	router.HandleFunc("/backfill_all_bookmarks", controllers.FixUpBookmarks).Methods("GET") //just meant for internal use - not for external use

	//POAP related stuff (some could be called client side directly but this protects the API key)
	router.HandleFunc("/get_poaps/{wallet}", controllers.GetPoapsByAddr).Methods("GET")
}

func initDB() {
	config :=
		database.Config{
			User:       "doadmin",
			Password:   os.Getenv("DB_PASSWORD"),
			ServerName: os.Getenv("DB_URL"),
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
