package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	/**
	websocketUpgrader is used to upgrade incomming HTTP requests into a persitent websocket connection
	*/
	websocketUpgrader = websocket.Upgrader{
		CheckOrigin:     checkOrigin,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

// Manager is used to hold references to all Clients Registered, and Broadcasting etc
type Manager struct {
	clients ClientList
	// chatrooms is the currently existing chatrooms
	chatrooms map[string]ClientList

	// Using a syncMutex here to be able to lcok state before editing clients
	// Could also use Channels to block
	sync.RWMutex
}

// NewManager is used to initalize all the values inside the manager
func NewManager() *Manager {
	return &Manager{
		clients:   make(ClientList),
		chatrooms: make(map[string]ClientList),
	}
}

// serveWS is a HTTP Handler that the has the Manager that allows connections
func (m *Manager) serveWS(w http.ResponseWriter, r *http.Request) {
	// Begin by upgrading
	log.Println("New connection")
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Create new Client
	client := NewClient(conn, m)
	// Register the Client to the Manager
	m.addClient(client)

	// Read messages from the client
	go client.readMessages()
	go client.writeMessages()
}

// addClient will push a new client and
// Place it in the default chatroom upon startup
func (m *Manager) addClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	m.clients[client] = true

	if _, ok := m.chatrooms[DEFAULT_CHATROOM]; !ok {
		m.chatrooms[DEFAULT_CHATROOM] = make(ClientList)
	}
	m.chatrooms[DEFAULT_CHATROOM][client] = true
	client.chatroom = DEFAULT_CHATROOM
}

// removeClient will remove the client and clean up
func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()
	// remove from chatroom
	if _, ok := m.chatrooms[client.chatroom]; ok {
		delete(m.chatrooms[client.chatroom], client)
	}
	// Check if client exists, then delete it and close egress
	if _, ok := m.clients[client]; ok {
		delete(m.clients, client)
		close(client.egress)
	}
}

// changeChatRoom is a event handler that changes the users current chatroom
func (m *Manager) changeChatRoom(event Event, c *Client) (*Event, error) {
	m.Lock()
	defer m.Unlock()

	log.Println(string(event.Payload))

	// Marshal Payload into wanted format
	var chatevent ChangeRoomEvent
	if err := json.Unmarshal(event.Payload, &chatevent); err != nil {
		return nil, fmt.Errorf("bad payload in request: %v", err)
	}

	if chatevent.Name == "" {
		return nil, errors.New("cannot use empty chatroom")
	}

	// remove from current chatroom
	if _, ok := m.chatrooms[c.chatroom]; ok {
		delete(m.chatrooms[c.chatroom], c)
	}
	// Change room and add it if it does not exist
	if _, ok := m.chatrooms[chatevent.Name]; !ok {
		m.chatrooms[chatevent.Name] = make(ClientList)
	}
	m.chatrooms[chatevent.Name][c] = true

	c.chatroom = chatevent.Name
	return nil, nil
}

func (m *Manager) sendMessage(event Event, c *Client) (*Event, error) {
	// Marshal Payload into wanted format
	log.Println("new message broadcasted")
	var chatevent SendMessageEvent
	if err := json.Unmarshal(event.Payload, &chatevent); err != nil {
		return nil, fmt.Errorf("bad payload in request: %v", err)
	}

	var broadMessage BroadcastMessageEvent

	broadMessage.Sent = time.Now()
	broadMessage.Message = chatevent.Message
	broadMessage.From = chatevent.From

	var outgoingEvent Event

	data, err := json.Marshal(broadMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal broadcast message: %v", err)
	}
	outgoingEvent.Payload = data
	outgoingEvent.Type = EventNewMessage
	// Broadcast to all other Clients in the same chatroom
	for client := range m.chatrooms[c.chatroom] {
		client.egress <- outgoingEvent
	}

	return nil, nil

}
