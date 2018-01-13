package main

import (
	"fmt"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"html/template"
	"log"
	"net/http"
	"os"
	"quickchat/database"
	"strconv"
)

var mainPage = template.Must(template.ParseFiles("template/index.html"))
var mainPage2 = template.Must(template.ParseFiles("template/index.html"))
var chatPage = template.Must(template.ParseFiles("template/chat.html"))

var db *gorm.DB
var hublist map[int]*Hub

func main() {
	// make migrations and delete all expired chats
	hublist = make(map[int]*Hub)
	database.Migrate()
	db = database.Connect()
	database.ChatDeleteExpired()

	// set log output file
	f, err := os.OpenFile("testlogfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("ERROR opening file")
	}

	log.SetOutput(f)

	defer func() {
		f.Close()
		db.Close()
	}()

	r := mux.NewRouter()
	r.PathPrefix("/static/").Handler( // serve static files
		http.StripPrefix("/static/", http.FileServer(http.Dir("./template/static/"))),
	)
	r.HandleFunc("/", ShowMain).Methods("GET")
	r.HandleFunc("/chat/duplicate", ShowMainDuplicateChat).Methods("GET") // main index page
	r.HandleFunc("/{Name}", ShowChat).Methods("GET")                      // chat page
	r.HandleFunc("/ws/{ID}/{username}", Chat)                             // websocket connection page
	r.HandleFunc("/chat/create", CreateChat).Methods("POST")              // create chat
	CSRF := csrf.Protect(database.RandStringBytes(), csrf.Secure(false))
	log.Println("Server running ... ")
	http.ListenAndServe("0.0.0.0:80", CSRF(r))
}

// Index page
func ShowMain(w http.ResponseWriter, r *http.Request) {
	mainPage.Execute(w, map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(r),
		"msg":            "Create your chat...",
		"NumberOfConnections": NumberOfConnections,
	})
}

func ShowMainDuplicateChat(w http.ResponseWriter, r *http.Request) {
	mainPage2.Execute(w, map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(r),
		"msg":            "Chat already exist...",
		"NumberOfConnections": NumberOfConnections,
	})
}

// Loads the chat page
func ShowChat(w http.ResponseWriter, r *http.Request) {
	var chat database.Chat
	vars := mux.Vars(r)
	Name := vars["Name"]

	if err := db.Where("Name = ? ", Name).First(&chat).Error; err != nil {
		http.Redirect(w, r, "/", 303)
		return
	}

	comments := []database.Comment{}
	db.Where("chat_id = ?", chat.ID).Order("created_at asc").Find(&comments)
	var data map[string]interface{}
	data = make(map[string]interface{})
	data["ChatName"] = chat.Name
	data["ID"] = chat.ID
	data["comments"] = comments
	data["csrfField"] = csrf.TemplateField(r)
	chatPage.Execute(w, data)
}

// Creates a new chat
func CreateChat(w http.ResponseWriter, r *http.Request) {
	ChatName := r.FormValue("chatName")
	key := r.FormValue("key")

	if len(ChatName) == 0 || len(key) == 0 {
		return
	}
	// create if does not exist
	if err := db.Where("Name = ?", ChatName).Find(&database.Chat{}).Error; err != nil {
		database.ChatCreate(ChatName, key)
		http.Redirect(w, r, "/"+ChatName, 303)
	} else {
		http.Redirect(w, r, "/chat/duplicate", 303)
	}
}

// websocket connect and verification
func Chat(w http.ResponseWriter, r *http.Request) {
	var users []database.User
	var chat database.Chat
	var key, userkey string
	vars := mux.Vars(r)
	ID, _ := strconv.Atoi(vars["ID"])
	username := vars["username"]
	keyString := r.URL.Query()["key"]
	userkeyString := r.URL.Query()["userkey"]

	// check if keys enetered are not empty
	if len(keyString) != 0 && len(userkeyString) != 0 {
		key = keyString[0]
		userkey = userkeyString[0]
		if len(key) == 0 || len(userkey) == 0 {
			return
		}
	} else {
		return
	}

	chat.ID = uint(ID)
	db.Model(&chat).Related(&users)
	var found bool
	log.Println("Trying to connenct username:", username, "passcode:", userkey)
	for _, user := range users {
		if ok, chat := VerifyKey(ID, key); user.Username == username && user.Skey == userkey && ok {
			connect(w, r, ID, username, key, chat) // if user key and Chat key matched
			return
		} else if user.Username == username {
			found = true // if user exists but wrong key
		}
	}

	// if user does not exist but Chat key is correct
	if ok, chat := VerifyKey(ID, key); !found && ok {
		log.Println("New User connected", username)
		database.UserCreate(ID, username, userkey, chat)
		connect(w, r, ID, username, key, chat)
	}

}

// create Hub
func connect(w http.ResponseWriter, r *http.Request, ID int, username, key string, chat database.Chat) {
	if _, ok := hublist[ID]; !ok { // if Hub does not exist in map for a chat
		hub := newHub(ID, key)
		go hub.run()
		serveWs(hub, username, w, r, chat)
		hublist[ID] = hub
	} else {
		serveWs(hublist[ID], username, w, r, chat)
	}
}

// Verify key given ID for the chat
func VerifyKey(ID int, key string) (bool, database.Chat) {
	var chat database.Chat

	if err := db.Where("ID = ? and skey = ?", ID, key).First(&chat).Error; err != nil {
		log.Println("Wrong key or ID")
		return false, database.Chat{}
	}
	return true, chat
}
