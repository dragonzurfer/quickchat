package main

import (
	"bytes"
	// "fmt"
	"github.com/gorilla/websocket"
	// "github.com/jinzhu/gorm"
	"log"
	"net/http"
	"quickchat/database"
	"strings"
	"time"
)

var NumberOfConnections int

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 2048
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type User struct {
	name      string
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	chat      database.Chat
	writekill chan bool
}

func (c *User) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		log.Println(c.hub.chatid, ": user", c.name, "sent a message")
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("%v: error: %v", err, c.hub.chatid)
			}
			log.Println(c.hub.chatid, ": read user", c.name, "going down :", err)
			c.writekill <- true // if error make sure to kill write as well
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		smessage := string(message)
		splitMessage := strings.Split(smessage, ":")
		if len(splitMessage) == 2 {
			if splitMessage[0] == c.name {
				database.CommentCreate(c.hub.chatid, c.name, splitMessage[1], c.chat)
				c.hub.broadcast <- message
			} else {
				c.writekill <- true
				return
			}
		}
	}
}

func (c *User) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		// c.hub.unregister <- c
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println(c.hub.chatid, ": write user", c.name, "going down")
				return
			}
		case <-c.writekill:
			log.Println(c.hub.chatid, ": write user", c.name, "going down")
			return
		}
	}
}

func serveWs(hub *Hub, name string, w http.ResponseWriter, r *http.Request, chat database.Chat) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	user := &User{name: name, hub: hub, conn: conn, chat: chat,
		send:      make(chan []byte, 256),
		writekill: make(chan bool),
	}
	user.hub.register <- user
	// start writer and reader on different go routines
	go user.writePump()
	go user.readPump()
	log.Println("User", user.name, " connected!")
}
