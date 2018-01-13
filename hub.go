package main

import (
	// "fmt"
	"log"
	"quickchat/database"
	"time"
)

type Hub struct {
	pass       string
	chatid     int
	users      map[*User]bool
	broadcast  chan []byte
	register   chan *User
	unregister chan *User
}

func newHub(id int, key string) *Hub {
	return &Hub{
		pass:       key,
		chatid:     id,
		users:      make(map[*User]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *User),
		unregister: make(chan *User),
	}
}

func (hub *Hub) run() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case newUser := <-hub.register:
			hub.users[newUser] = true
			NumberOfConnections += 1

		case delUser := <-hub.unregister:
			if _, ok := hub.users[delUser]; ok {
				delete(hub.users, delUser)
				close(delUser.send)
				delUser.conn.Close()
				log.Println("Killed user", delUser.name)
				NumberOfConnections -= 1
				// if hub has no connections, close hub
				if len(hub.users) == 0 {
					log.Println("Hub", hub.chatid, "going down")
					delete(hublist, hub.chatid)
					return
				}
			}

		case message := <-hub.broadcast:
			for user := range hub.users {
				select {
				case user.send <- message:
					log.Println("Got message in ", hub.chatid)
				default:
					close(user.send)
					delete(hub.users, user)
				}
			}

		case <-ticker.C:
			// if Chat expired, close the hub
			if !database.ChatExists(hub.chatid) {
				log.Println(hub.chatid, ": Closing all connections")
				for user := range hub.users {
					user.conn.Close()
					close(user.send)
					delete(hub.users, user)
					log.Println("Killed user", user.name)
				}
				delete(hublist, hub.chatid)
				return
			}
		}

	}
}
