package realtime

import (
	"api/models"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	competitionClients = make(map[string]map[*websocket.Conn]bool) // Map of competition ID to connected clients
	broadcast           = make(chan TryUpdate)                     // Broadcast channel for updates
	mutex               sync.Mutex                                 // Mutex to protect competitionClients map
)

// TryUpdate represents a new or updated try
type TryUpdate struct {
	CompetitionID string      `json:"competition_id"`
	Try           models.Try  `json:"try"`
	UpdateType    string      `json:"update_type"` // "new" or "update"
}

// RegisterClient adds a WebSocket client to a specific competition
func RegisterClient(competitionID string, conn *websocket.Conn) {
	mutex.Lock()
	if competitionClients[competitionID] == nil {
		competitionClients[competitionID] = make(map[*websocket.Conn]bool)
	}
	competitionClients[competitionID][conn] = true
	mutex.Unlock()
}

// UnregisterClient removes a WebSocket client from a specific competition
func UnregisterClient(competitionID string, conn *websocket.Conn) {
	mutex.Lock()
	if clients, exists := competitionClients[competitionID]; exists {
		delete(clients, conn)
		if len(clients) == 0 {
			delete(competitionClients, competitionID)
		}
	}
	mutex.Unlock()
}

// BroadcastTryUpdate sends updates to all clients connected to a specific competition
func BroadcastTryUpdate(update TryUpdate) {
	broadcast <- update
}

func handleBroadcast() {
	for {
		update := <-broadcast
		mutex.Lock()
		if clients, exists := competitionClients[update.CompetitionID]; exists {
			for client := range clients {
				if err := client.WriteJSON(update); err != nil {
					log.Printf("WebSocket write error: %v", err)
					client.Close()
					delete(clients, client)
				}
			}
		}
		mutex.Unlock()
	}
}

func init() {
	go handleBroadcast()
}