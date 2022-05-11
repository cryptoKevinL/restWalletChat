package database

import (
	"log"
	"rest-go-demo/entity"

	"github.com/jinzhu/gorm"
)

//Connector variable used for CRUD operation's
var Connector *gorm.DB

//Connect creates MySQL connection
func Connect(connectionString string) error {
	var err error
	Connector, err = gorm.Open("mysql", connectionString)
	if err != nil {
		return err
	}
	log.Println("Connection was successful!!")
	return nil
}

//Migrate create/updates database table
func Migrate(table *entity.Inbox) {
	Connector.AutoMigrate(&table)
	log.Println("Inboxes migrated")
}
func MigrateChatItem(table *entity.ChatItem) {
	Connector.AutoMigrate(&table)
	log.Println("ChatItems migrated")
}
