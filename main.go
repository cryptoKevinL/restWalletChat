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
	router.HandleFunc("/get_inbox/{address}", controllers.GetInboxByOwner).Methods("GET")
	router.HandleFunc("/create_chatitem", controllers.CreateChatitem).Methods("POST")
	router.HandleFunc("/getall_chatitems", controllers.GetAllChatitems).Methods("GET")
	router.HandleFunc("/get_unread_cnt/{address}", controllers.GetUnreadMsgCntTotal).Methods("GET")
	router.HandleFunc("/get_unread_cnt/{fromaddr}/{toaddr}", controllers.GetUnreadMsgCnt).Methods("GET")
	router.HandleFunc("/getall_chatitems/{address}", controllers.GetChatFromAddress).Methods("GET")
	router.HandleFunc("/getnft_chatitems/{address}", controllers.GetChatNftAllItemsFromAddr).Methods("GET")
	router.HandleFunc("/getnft_chatitems/{nftaddr}/{nftid}", controllers.GetChatNftContext).Methods("GET")
	router.HandleFunc("/update_chatitem/{fromaddr}/{toaddr}", controllers.UpdateChatitemByOwner).Methods("PUT")
	router.HandleFunc("/deleteall_chatitems/{address}", controllers.DeleteAllChatitemsToAddressByOwner).Methods("DELETE")
	router.HandleFunc("/create_settings", controllers.CreateSettings).Methods("POST")
	router.HandleFunc("/update_settings", controllers.UpdateSettings).Methods("PUT")
	router.HandleFunc("/get_settings/{address}", controllers.GetSettings).Methods("GET")
	router.HandleFunc("/delete_settings/{address}", controllers.DeleteSettings).Methods("DELETE")
	router.HandleFunc("/create_comments", controllers.CreateComments).Methods("POST")
	router.HandleFunc("/get_comments", controllers.GetAllComments).Methods("GET")
	router.HandleFunc("/get_comments/{nftaddr}/{nftid}", controllers.GetComments).Methods("GET")
	router.HandleFunc("/delete_comments/{fromaddr}/{nftaddr}/{nftid}", controllers.DeleteComments).Methods("DELETE")
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
	// database.Migrate(&entity.Settings{})
	//database.MigrateComments(&entity.Comments{})
	// database.MigrateChatitem(&entity.Chatitem{})
}
