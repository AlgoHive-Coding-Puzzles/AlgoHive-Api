package competitions

import (
	"api/realtime"
	"api/services"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// CompetitionWebSocket handles WebSocket connections for a specific competition
func CompetitionWebSocket(c *gin.Context) {
	competitionID := c.Param("id")

	// Validate competition ID
	if !services.CompetitionExists(competitionID) {
		c.JSON(404, gin.H{"error": "Competition not found"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	realtime.RegisterClient(competitionID, conn)
	defer func() {
		realtime.UnregisterClient(competitionID, conn)
		conn.Close()
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}
	}
}