package database

import (
	// "fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"log"
	"time"
)

type Chat struct {
	gorm.Model
	Name string `gorm:"not null;unique"`
	Skey string `gorm:"size:500"`
}

type User struct {
	Chat     Chat
	ChatID   int    `gorm:"ForeignKey:ChatId"`
	Skey     string `gorm:"not null"`
	Username string `gorm:"not null"`
}

type Comment struct {
	gorm.Model
	Username string `gorm:"not null"`
	Message  string `gorm:"not null"`
	Chat     Chat
	ChatID   int `gorm:"ForeignKey:ChatId"`
}

func Migrate() {
	db := Connect()
	defer db.Close()
	db.AutoMigrate(&Chat{}, &Comment{}, &User{})
	log.Println("Migrations Complete!")
}

// Create chat
func ChatCreate(chatName, chatkey string) {
	NewChat := Chat{
		Name: chatName,
		Skey: chatkey,
	}
	db := Connect()
	defer db.Close()
	db.Save(&NewChat)

	// Delete after 24 hours
	go func(NewChat *Chat) {

		defer func() {
			ChatDelete(NewChat.ID)
		}()

		time.Sleep(24 * time.Hour)

	}(&NewChat)

	log.Println(NewChat.Name, " chat was created!")
}

// Delete chat
func ChatDelete(id uint) {
	db := Connect()
	defer db.Close()
	var chat Chat
	chat.ID = id
	err0 := db.Unscoped().Delete(&chat).Error
	err1 := db.Where("chat_id = ?", id).Delete(&User{}).Error
	err2 := db.Unscoped().Where("chat_id = ?", id).Delete(&Comment{}).Error

	if err0 != nil || err1 != nil || err1 != nil {
		log.Println("error deleting chat ", err0, err1, err2)
		return
	}
	log.Println("Deleted chat", id)
}

// Create user
func UserCreate(id int, Name, key string, chat Chat) {
	NewUser := User{
		Username: Name,
		Skey:     key,
		Chat:     chat,
	}
	db := Connect()
	defer db.Close()
	db.Save(&NewUser)

	log.Println(NewUser.Username, " user was created!")
}

// Create comment
func CommentCreate(ID int, username, smessage string, chat Chat) {
	db := Connect()
	defer db.Close()
	comment := Comment{
		Username: username,
		Message:  smessage,
		Chat:     chat,
	}
	db.Save(&comment)
	log.Println("New comment saved to ", ID)
}

// Check if a chat exists
func ChatExists(ID int) bool {
	db := Connect()
	defer db.Close()
	var chat Chat
	if err := db.Where("ID = ?", ID).First(&chat).Error; err != nil {
		return false
	}
	return true
}

// Delete chats that existed for more than 24 hours
func ChatDeleteExpired() {
	db := Connect()
	defer db.Close()

	var chats []Chat
	db.Find(&chats)

	for _, chat := range chats {
		timediff := time.Now().Sub(chat.CreatedAt)
		if timediff >= 24*time.Hour {
			ChatDelete(chat.ID)
		}
	}
}
