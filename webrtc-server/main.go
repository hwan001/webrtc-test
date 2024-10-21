package main

import (
	"fmt"
	"net/http"
	"time"

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

	// Set a Ping handler to respond to Ping messages from the client
	ws.SetPingHandler(func(appData string) error {
		fmt.Printf("Received Ping message from client: %v, responding with Pong...\n", ws.RemoteAddr())

		// Set a write deadline for the Pong message
		ws.SetWriteDeadline(time.Now().Add(10 * time.Second))

		// Send the Pong message
		return ws.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
	})

	// Register the new client
	clients[ws] = true
	fmt.Printf("New client connected: %v\n", ws.RemoteAddr())

	for {
		var msg []byte
		// Read a message from the WebSocket connection
		_, msg, err := ws.ReadMessage()
		if err != nil {
			fmt.Printf("Client disconnected: %v, Error: %v\n", ws.RemoteAddr(), err)
			delete(clients, ws)
			break
		}
		fmt.Printf("Received message from client: %v\n", ws.RemoteAddr())

		// Send the message to the broadcast channel
		broadcast <- msg
	}
}

// Handles incoming messages and broadcasts them to all connected clients
func handleMessages() {
	for {
		// Grab the message from the broadcast channel
		msg := <-broadcast
		fmt.Println("Broadcasting message to all clients...")

		// Send the message to every connected client
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				fmt.Printf("Failed to send message to client: %v, Error: %v\n", client.RemoteAddr(), err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}