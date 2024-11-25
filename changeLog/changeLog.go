package changeLog

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type LogEntry struct {
	ID        int    `json:"id"`
	Change    string `json:"change"`
	Timestamp string `json:"timestamp"`
}

var db *sql.DB
var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan LogEntry)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func Setup(database *sql.DB) {
	db = database
	go handleBroadcast()
}

func GetChangeLog(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("perPage", "100"))

	if page < 1 {
		page = 1
	}

	offset := (page - 1) * perPage

	entries := []LogEntry{}

	query := "SELECT id, change, timestamp FROM change_log ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	rows, err := db.Query(query, perPage, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error retrieving log entries.",
			"error":   err.Error(),
		})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var entry LogEntry
		if err := rows.Scan(&entry.ID, &entry.Change, &entry.Timestamp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Error retrieving log entries.",
				"error":   err.Error(),
			})
			return
		}
		entries = append(entries, entry)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Log entries retrieved successfully.",
		"data":    entries,
	})
}

func AddEntryToLog(change string) error {
	timestamp := strconv.FormatInt(time.Now().Unix()*1000, 10)
	_, err := db.Exec("INSERT INTO change_log (change, timestamp) VALUES (?, ?)", change, timestamp)
	if err != nil {
		log.Printf("Failed to add log entry: %v", err)
		return err
	}

	entry := LogEntry{
		Change:    change,
		Timestamp: timestamp,
	}
	broadcast <- entry

	return nil
}

func CreateTable() {
	createTableSQL := `CREATE TABLE IF NOT EXISTS change_log (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        change TEXT NOT NULL,
        timestamp TEXT NOT NULL
    );`

	_, err := db.Exec(createTableSQL)
	if err != nil {
		log.Fatal("Failed to create change_log table: ", err)
	}
}

func ChangeLogWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	clients[conn] = true

	sendCurrentChangeLogs(conn)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket connection closed: %v", err)
			delete(clients, conn)
			break
		}
	}
}

func sendCurrentChangeLogs(conn *websocket.Conn) {
	entries := []LogEntry{}
	query := "SELECT id, change, timestamp FROM change_log ORDER BY timestamp DESC"
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error retrieving change logs: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var entry LogEntry
		if err := rows.Scan(&entry.ID, &entry.Change, &entry.Timestamp); err != nil {
			log.Printf("Error scanning log entry: %v", err)
			return
		}
		entries = append(entries, entry)
	}

	log.Printf("Sending change logs to WebSocket client: %+v", entries)

	err = conn.WriteJSON(entries)
	if err != nil {
		log.Printf("Failed to send current logs to WebSocket client: %v", err)
	}
}

func handleBroadcast() {
	for {
		entry := <-broadcast

		for client := range clients {
			err := client.WriteJSON(entry)
			if err != nil {
				log.Printf("Failed to send message to WebSocket client: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
