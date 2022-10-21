package main

import (
	"log"
	"net/http"
	"os"
	"rest-go-demo/auth"
	"rest-go-demo/controllers"
	"rest-go-demo/database"

	"github.com/joho/godotenv"
	"github.com/rs/cors"

	_ "rest-go-demo/docs" // docs is generated by Swag CLI, you have to import it

	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/gorilla/mux"
	_ "github.com/jinzhu/gorm/dialects/mysql" //Required for MySQL dialect

	"time"
)

// @title WalletChat API
// @version 1.0
// @description Wecome to the WalletChat API Documentation
// @description
// @description Please make note that some JSON data structures are shared for both input/output.
// @description Required input parameters will have a red * next to them in the data type outline at
// @description the bottom of the page, along with a comment.  This means when executing API functionality
// @description from this API page, some fields may need to be removed from the JSON struct before submitting.
// @description Please email the developers with any issues.
// @description Some JSON data structures are output only, and will be marked as such as well.
// @description
// @description v1 includes JWT Authentication
// @description except for AUTH functions, all /v1 endpoints must include "Bearer <JWT>" in all requests showing the Lock Icon"
// @description For this API Doc, use the "Authorize" button on the right hand side to ender "Bearer <JWT>" where the JWT will
// @description come from the return value of the /signin endpoint.  Please read the /register, /users/<>/nonce, and /signin
// @description descriptions to understand the login workflow via JWT Auth.
// @description
// @description v1.1 will include encyrption for DMs, using LIT Protocol for encryption
// @description
// @wallet_chat API Support via Twitter
// @contact.url https://walletchat.fun
// @contact.email walletchatextension@gmail.com
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @host restwalletchat-app-sey3k.ondigitalocean.app
// @BasePath
func main() {
	godotenv.Load(".env")

	// from := mail.NewEmail("NF3 Notifications", "contact@walletchat.fun")
	// subject := "Message Waiting In WalletChat"
	// to := mail.NewEmail("xrpMaxi", "savemynft@gmail.com")
	// plainTextContent := "You have message from vitalik.eth waiting in WalletChat, please login via the app direct to read!"
	// htmlContent := "<strong>You have message from vitalik.eth waiting in WalletChat, please login via the app direct to read!</strong>"
	// message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	// client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	// response, err := client.Send(message)
	// if err != nil {
	// 	log.Println(err)
	// } else {
	// 	fmt.Println(response.StatusCode)
	// 	fmt.Println(response.Body)
	// 	fmt.Println(response.Headers)
	// }

	initDB()
	log.Println("Starting the HTTP server on port 8080")

	jwtProvider := auth.NewJwtHmacProvider(
		os.Getenv("JWT_HMAC_SECRET"),
		"https://walletchat.fun",
		time.Minute*60*24*30,
	)

	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/register", auth.RegisterHandler()).Methods("POST")
	router.HandleFunc("/users/{address}/nonce", auth.UserNonceHandler()).Methods("GET")
	router.HandleFunc("/signin", auth.SigninHandler(jwtProvider)).Methods("POST")
	router.PathPrefix("/docs").Handler(httpSwagger.WrapHandler)

	wsRouter := router.PathPrefix("/v1").Subrouter()

	wsRouter.Use(auth.AuthMiddleware(jwtProvider))
	wsRouter.HandleFunc("/welcome", auth.WelcomeHandler()).Methods("GET")

	initaliseHandlers(wsRouter)

	//handler := cors.Default().Handler(router) //cors.AllowAll().Handler(router)
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"https://app.walletchat.fun", "http://localhost:3000", "http://localhost:8080", "https://v1.walletchat.fun"},
		AllowCredentials: true,
		// Enable Debugging for testing, consider disabling in production
		//Debug: true,
	})
	handler := c.Handler(router)
	log.Fatal(http.ListenAndServe(":8080", handler))
}

func initaliseHandlers(router *mux.Router) {
	//1-to-1 chats (both general and NFT related)
	router.HandleFunc("/get_unread_cnt/{address}", controllers.GetUnreadMsgCntTotal).Methods("GET")
	router.HandleFunc("/get_unread_cnt_by_type/{address}/{type}", controllers.GetUnreadMsgCntTotalByType).Methods("GET")
	router.HandleFunc("/get_unread_cnt/{fromaddr}/{toaddr}", controllers.GetUnreadMsgCnt).Methods("GET")
	router.HandleFunc("/get_unread_cnt/{address}/{nftaddr}/{nftid}", controllers.GetUnreadMsgCntNft).Methods("GET")
	router.HandleFunc("/get_unread_cnt_nft/{address}", controllers.GetUnreadMsgCntNftAllByAddr).Methods("GET")
	router.HandleFunc("/getall_chatitems/{address}", controllers.GetChatFromAddress).Methods("GET")
	router.HandleFunc("/getall_chatitems/{fromaddr}/{toaddr}", controllers.GetAllChatFromAddressToAddr).Methods("GET")
	router.HandleFunc("/getread_chatitems/{fromaddr}/{toaddr}", controllers.GetReadChatFromAddressToAddr).Methods("GET")
	router.HandleFunc("/getall_chatitems/{fromaddr}/{toaddr}/{time}", controllers.GetNewChatFromAddressToAddr).Methods("GET")
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
	router.HandleFunc("/unreadcount/{address}", controllers.GetUnreadcnt).Methods("GET", "OPTIONS")
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
	//router.HandleFunc("/name", controllers.UpdateAddrNameItem).Methods("PUT")
	router.HandleFunc("/name/{address}", controllers.GetAddrNameItem).Methods("GET")

	//Logos / Images stored in base64
	router.HandleFunc("/image", controllers.CreateImageItem).Methods("POST")
	router.HandleFunc("/image", controllers.UpdateImageItem).Methods("PUT")
	router.HandleFunc("/image/{name}", controllers.GetImageItem).Methods("GET")

	//settings items - currently this is the public key added upon first login for encryption/signing without MM
	//router.HandleFunc("/create_settings", controllers.CreateSettings).Methods("POST")
	router.HandleFunc("/update_settings", controllers.UpdateSettings).Methods("POST")
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
