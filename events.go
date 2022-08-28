package main

import (
	"encoding/json"
	"time"
)

const (
	EventChangeRoom  = "change_room"
	EventSendMessage = "send_message"
	EventNewMessage  = "new_message"
)

// EventHandler is a function signature that is used to affect messages on the socket and triggered
// depending on the type
type EventHandler func(event Event, c *Client) (*Event, error)

var (
	/**
	EventHandlers is a map containing handlers for different Event types
	*/
	eventHandlers map[string]EventHandler
)

// Event is the Messages sent over the websocket
// Used to differ between different actions
type Event struct {
	// Type is the message type sent
	Type string `json:"type"`
	// Payload is the data Based on the Type
	Payload json.RawMessage `json:"payload"`
}

type ChangeRoomEvent struct {
	Name string `json:"name"`
}

type SendMessageEvent struct {
	Message string `json:"message"`
	From    string `json:"from"`
}

type BroadcastMessageEvent struct {
	SendMessageEvent
	Sent time.Time `json:"sent"`
}
