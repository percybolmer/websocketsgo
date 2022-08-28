package main

import (
	"log"
	"net/http"
)

func main() {
	setupAPI()

	// Serve on port :8080, fudge yeah hardcoded port
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// setupAPI will start all Routes and their Handlers
func setupAPI() {
	// Create new Connection manager
	manager := NewManager()

	setupEventHandlers(manager)
	// Serve the ./frontend directory at Route /
	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/ws", manager.serveWS)
}

// setupEventHandlers configures and adds all data and setups needed stuff for eventhandlers
func setupEventHandlers(m *Manager) {
	eventHandlers = make(map[string]EventHandler)
	eventHandlers[EventChangeRoom] = m.changeChatRoom
	eventHandlers[EventSendMessage] = m.sendMessage
}

// checkOrigin will check origin and return true if its true
func checkOrigin(r *http.Request) bool {

	// Check the request origin
	origin := r.Header.Get("Origin")

	switch origin {
	case "http://localhost:8080":
		return true
	default:
		return false
	}
}
