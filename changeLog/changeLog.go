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

	err = conn.WriteJSON(entries)
	if err != nil {
		log.Printf("Failed to send current logs to WebSocket client: %v", err)
	}
}

func handleBroadcast() {
	for {
		<-broadcast

		entries := []LogEntry{}
		query := "SELECT id, change, timestamp FROM change_log ORDER BY timestamp DESC"
		rows, err := db.Query(query)
		if err != nil {
			log.Printf("Error retrieving change logs: %v", err)
			continue
		}
		// defer rows.Close()

		for rows.Next() {
			var logEntry LogEntry
			if err := rows.Scan(&logEntry.ID, &logEntry.Change, &logEntry.Timestamp); err != nil {
				log.Printf("Error scanning log entry: %v", err)
				continue
			}
			entries = append(entries, logEntry)
		}

		if err := rows.Err(); err != nil {
			log.Printf("Error with rows iteration: %v", err)
			continue
		}

		for client := range clients {
			err := client.WriteJSON(entries)
			if err != nil {
				log.Printf("Failed to send full logs to WebSocket client: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
