package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

var (
	ErrEventNotSupported = errors.New("event type is not supported")
)

const (
	DEFAULT_BUFFER_LIMIT = 255
	DEFAULT_CHATROOM     = "general"
)

var (
	// pongWait is how long we will await a pong response from client
	pongWait = 10 * time.Second
	// pingInterval has to be less than pongWait, We cant multiply by 0.9 to get 90% of time
	// Because that can make decimals, so instead *9 / 10 to get 90%
	// The reason why it has to be less than PingRequency is becuase otherwise it will send a new Ping before getting response
	pingInterval = (pongWait * 9) / 10

	// maximumMessageSize allowed per Client
	maximumMessageSize int64 = 512
	// writeTimeout is how long a write is allowed to take
	writeTimeout = 10 * time.Second
)

// ClientList is a map used to help manage a map of clients
type ClientList map[*Client]bool

// Client is a websocket client, basically a frontend visitor
type Client struct {
	// the websocket connection
	connection *websocket.Conn

	// manager is the manager used to manage the client
	manager *Manager

	// egress is outgoing messages from the Client
	// We use a egress Channel to ensure there is only 1 concurrent Writer in the Application on the Connection
	// This is a buffered Channel
	egress chan Event

	// chatroom is the currently selected chatroom
	chatroom string
}

// NewClient is used to initialize a new Client with all required values initialized
func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		connection: conn,
		manager:    manager,
		egress:     make(chan Event, DEFAULT_BUFFER_LIMIT),
		chatroom:   DEFAULT_CHATROOM,
	}
}

// readMessages will start the client to read messages and handle them
// appropriatly.
// This is suppose to be ran as a goroutine
func (c *Client) readMessages() {
	defer func() {
		// Close Connection once we shutdown
		//c.connection.Close()
		c.manager.removeClient(c)
	}()
	// Configure Message Size
	c.connection.SetReadLimit(maximumMessageSize)
	// Configure Wait time for Pong response, use Current time + Wait Period
	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println(err)
		return
	}
	c.connection.SetPongHandler(c.pongHandler)

	// For loops repeadly forever until broken
	for {
		messagetype, message, err := c.connection.ReadMessage()
		if err != nil {
			// If Connection is closed, we will Recieve an error here
			// We only want to log Strange errors, but simple Disconnection
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}
			break // Break the loop to close conn & Cleanup
		}
		log.Println("MessageType: ", messagetype)

		// Recieve on the application defined standard
		var request Event
		if err := json.Unmarshal(message, &request); err != nil {
			log.Printf("error marshalling message: %v", err)
			break // Breaking the connection here might be harsh xD
		}

		if err := c.handleEvent(request); err != nil {
			log.Println(err)
		}

	}
}

// pongHandler is used to handle PongMessages for the Client
func (c *Client) pongHandler(pongMsg string) error {
	// Current time + Pong Wait time
	log.Println("pong")
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}

func (c *Client) writeMessages() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.connection.Close()
	}()

	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				// Manager has closed this connection channel
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					// This
					log.Println("connection closed: ", err)
				}
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Println(err)
				return // closes the connection, should we really
			}
			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Println(err)
			}
			log.Println("send message")
		case <-ticker.C:
			log.Println("ping")

			if err := c.connection.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Println("writemsg: ", err)
				return // return to break this goroutine triggeing cleanup
			}
		}

	}
}

// handleEvent is used to handle the incomming event
func (c *Client) handleEvent(event Event) error {
	// Execute eventhandler if present in event map
	if handler, ok := eventHandlers[event.Type]; ok {
		// execute selected handler
		responseEvent, err := handler(event, c)
		if err != nil {
			return fmt.Errorf("failed to execute handler: %v", err)
		}

		// if response is empty, skip it
		if responseEvent == nil {
			return nil
		}
		data, err := json.Marshal(responseEvent)
		if err != nil {
			return fmt.Errorf("failed to marshal response: %v", err)
		}
		// Send the response to the Client
		if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
			return fmt.Errorf("failed to respond to client: %v", err)
		}
		return nil
	} else {
		return ErrEventNotSupported
	}
}
