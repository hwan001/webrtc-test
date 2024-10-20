package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

// WebSocket upgrader to upgrade HTTP connections to WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// A map to keep track of connected clients (browser and agent)
var clients = make(map[*websocket.Conn]bool)

// A channel to broadcast messages to all clients
var broadcast = make(chan []byte)

func main() {
	// Start the WebSocket server
	http.HandleFunc("/signal", handleConnections)
	go handleMessages()

	fmt.Println("Signaling Server started at :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}

// Handles new WebSocket connections
func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a WebSocket connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Failed to upgrade connection: %v\n", err)
		return
	}
	defer ws.Close()

	// Register the new client
	clients[ws] = true

	for {
		var msg []byte
		// Read a message from the WebSocket connection
		_, msg, err := ws.ReadMessage()
		if err != nil {
			delete(clients, ws)
			break
		}

		// Send the message to the broadcast channel
		broadcast <- msg
	}
}

// Handles incoming messages and broadcasts them to all connected clients
func handleMessages() {
	for {
		// Grab the message from the broadcast channel
		msg := <-broadcast

		// Send the message to every connected client
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				client.Close()
				delete(clients, client)
			}
		}
	}
}
