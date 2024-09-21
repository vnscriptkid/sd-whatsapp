package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Message struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Content string `json:"content"`
}

var clients = make(map[string]*websocket.Conn)
var clientsMutex sync.Mutex

var rdb *redis.Client
var ctx = context.Background()

func main() {
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Change this based on your Redis setup
	})

	go subscribeToChatChannel()

	router := gin.Default()
	router.GET("/ws", handleWebSocket)

	router.Run(":8081")
}

func handleWebSocket(c *gin.Context) {
	userID := c.Query("userId")
	if userID == "" {
		log.Printf("User ID not provided")
		return
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}
	defer ws.Close()

	addClient(userID, ws)

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			removeClient(userID)
			return
		}

		var message Message
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Printf("JSON unmarshal error: %v", err)
			continue
		}

		message.From = userID

		sendMessage(message)
	}
}

func addClient(userID string, conn *websocket.Conn) {
	clientsMutex.Lock()
	clients[userID] = conn
	clientsMutex.Unlock()
}

func removeClient(userID string) {
	clientsMutex.Lock()
	delete(clients, userID)
	clientsMutex.Unlock()
}

func sendMessage(message Message) {
	clientsMutex.Lock()
	toConn, exists := clients[message.To]
	clientsMutex.Unlock()

	if exists {
		sendDirectMessage(toConn, message)
	} else {
		publishMessage(message)
	}
}

func sendDirectMessage(conn *websocket.Conn, message Message) {
	msgBytes, _ := json.Marshal(message)
	if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func publishMessage(message Message) {
	msgBytes, _ := json.Marshal(message)
	if err := rdb.Publish(ctx, "chat", msgBytes).Err(); err != nil {
		log.Printf("Redis publish error: %v", err)
	}
}

func subscribeToChatChannel() {
	pubsub := rdb.Subscribe(ctx, "chat")
	ch := pubsub.Channel()

	for msg := range ch {
		var message Message
		if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
			log.Printf("JSON unmarshal error: %v", err)
			continue
		}

		clientsMutex.Lock()
		toConn, exists := clients[message.To]
		clientsMutex.Unlock()

		if exists {
			sendDirectMessage(toConn, message)
		}
	}
}
