package changeLog

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type LogEntry struct {
	ID        int    `json:"id"`
	Change    string `json:"change"`
	Timestamp string `json:"timestamp"`
}

var db *sql.DB

func Setup(database *sql.DB) {
	db = database
}

func GetChangeLog(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("perPage", "100"))

	if page < 1 {
		page = 1
	}

	offset := (page - 1) * perPage

	entries := []LogEntry{}

	query := "SELECT id, change, timestamp FROM change_log LIMIT ? OFFSET ?"
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
	_, err := db.Exec("INSERT INTO change_log (change, timestamp) VALUES (?, datetime('now'))", change)
	if err != nil {
		log.Printf("Failed to add log entry: %v", err)
		return err
	}
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
