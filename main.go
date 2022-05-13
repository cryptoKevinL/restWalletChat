package main

import (
	"log"
	"net/http"
	"rest-go-demo/controllers"
	"rest-go-demo/database"
	"rest-go-demo/entity"

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
	//router.HandleFunc("/create_inbox", controllers.CreateInbox).Methods("POST")
	//router.HandleFunc("/get_inbox", controllers.GetAllInbox).Methods("GET")
	router.HandleFunc("/get_inbox/{address}", controllers.GetInboxByOwner).Methods("GET")
	//router.HandleFunc("/update_inbox/{address}", controllers.UpdateInboxByOwner).Methods("PUT")
	//router.HandleFunc("/delete_inbox/{address}", controllers.DeleteInboxByOwner).Methods("DELETE")
	router.HandleFunc("/create_chatitem", controllers.CreateChatitem).Methods("POST")
	router.HandleFunc("/getall_chatitems", controllers.GetAllChatitems).Methods("GET")
	router.HandleFunc("/getall_chatitems/{address}", controllers.GetChatFromAddress).Methods("GET")
	router.HandleFunc("/update_chatitem/{fromaddr}/{toaddr}", controllers.UpdateChatitemByOwner).Methods("PUT")
	router.HandleFunc("/deleteall_chatitems/{fromaddr}", controllers.DeleteAllChatitemsToAddressByOwner).Methods("DELETE")
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
	//database.Migrate(&entity.Inbox{})
	database.MigrateChatitem(&entity.Chatitem{})
}
