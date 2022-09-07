package main

import (
	"log"

	"github.com/gorilla/websocket"
)

// ClientList is a map used to help manage a map of clients
type ClientList map[*Client]bool

// Client is a websocket client, basically a frontend visitor
type Client struct {
	// the websocket connection
	connection *websocket.Conn

	// manager is the manager used to manage the client
	manager *Manager
}

// NewClient is used to initialize a new Client with all required values initialized
func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		connection: conn,
		manager:    manager,
	}
}

// readMessages will start the client to read messages and handle them
// appropriatly.
// This is suppose to be ran as a goroutine
func (c *Client) readMessages() {
	defer func() {
		// Graceful Close the Connection once this
		// function is done
		c.manager.removeClient(c)
	}()
	// Loop Forever
	for {
		// ReadMessage is used to read the next message in queue
		// in the connection
		messageType, payload, err := c.connection.ReadMessage()

		if err != nil {
			// If Connection is closed, we will Recieve an error here
			// We only want to log Strange errors, but simple Disconnection
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}
			break // Break the loop to close conn & Cleanup
		}
		log.Println("MessageType: ", messageType)
		log.Println("Payload: ", string(payload))
	}
}

// func (c *Client) writeMessages() {
// 	defer func() {
// 		c.connection.Close()
// 	}()

// 	for {
// 		select {
// 		case message, ok := <-c.egress:
// 			if !ok {
// 				// Manager has closed this connection channel
// 				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
// 					// This
// 					log.Println("connection closed: ", err)
// 				}
// 				return
// 			}

// 			data, err := json.Marshal(message)
// 			if err != nil {
// 				log.Println(err)
// 				return // closes the connection, should we really
// 			}
// 			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
// 				log.Println(err)
// 			}
// 			log.Println("send message")
// 		}

// 	}
// }
