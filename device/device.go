package device

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Device struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	IsOn bool   `json:"isOn"`
}

var db *sql.DB

func Setup(database *sql.DB) {
	db = database
}

func ListDevices(c *gin.Context) {
	// Parse query parameters for pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("perPage", "100"))

	if page < 1 {
		page = 1
	}

	offset := (page - 1) * perPage

	// Query to count total items
	var total int
	countQuery := "SELECT COUNT(*) FROM devices"
	err := db.QueryRow(countQuery).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Error retrieving resources.", "error": err.Error()})
		return
	}

	// Adjust totalPages to ensure it always has a value
	totalPages := total / perPage
	if total%perPage != 0 {
		totalPages++
	}

	// Initialize devices as an empty slice
	devices := []Device{}

	// Query to fetch paginated items
	query := "SELECT id, name, type, isOn FROM devices LIMIT ? OFFSET ?"
	rows, err := db.Query(query, perPage, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Error retrieving resources.", "error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var u Device
		if err := rows.Scan(&u.ID, &u.Name, &u.Type, &u.IsOn); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Error retrieving resources.", "error": err.Error()})
			return
		}
		devices = append(devices, u)
	}

	// Construct the response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Resources retrieved successfully.",
		"data": gin.H{
			"items":       devices,
			"total":       total,
			"perPage":     perPage,
			"currentPage": page,
			"totalPages":  totalPages,
		},
	})
}

func AddDevice(c *gin.Context) {
	var newDevice Device
	if err := c.BindJSON(&newDevice); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Error creating resource.",
			"error":   err.Error(),
		})
		return
	}

	result, err := db.Exec("INSERT INTO devices (name, type, isOn) VALUES (?, ?, ?)", newDevice.Name, newDevice.Type, newDevice.IsOn)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error creating resource.",
			"error":   err.Error(),
		})
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error creating resource.",
			"error":   err.Error(),
		})
		return
	}

	newDevice.ID = int(id)
	newDevice.IsOn = bool(false)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Resource created successfully.",
		"data":    newDevice,
	})
}

func EditDevice(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid device ID.", "error": err.Error()})
		return
	}

	var updatedDevice Device
	if err := c.BindJSON(&updatedDevice); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Error updating resource.", "error": err.Error()})
		return
	}

	_, err = db.Exec("UPDATE devices SET name = ?, type = ?, isOn = ? WHERE id = ?", updatedDevice.Name, updatedDevice.Type, updatedDevice.IsOn, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Error updating resource.", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Resource updated successfully.", "data": updatedDevice})
}

func DeleteDevice(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid device ID.", "error": err.Error()})
		return
	}

	_, err = db.Exec("DELETE FROM devices WHERE id = ?", id)
	if err != nil { // Assuming 'err' holds the error from your delete operation
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Error deleting resource.", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Resource deleted successfully."})
}

// CreateTable creates the devices table if it does not exist
func CreateTable() {
	createTableSQL := `CREATE TABLE IF NOT EXISTS devices (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        type TEXT NOT NULL,
        isOn BOOLEAN NOT NULL DEFAULT 0
    );`

	_, err := db.Exec(createTableSQL)
	if err != nil {
		log.Fatal("Failed to create table: ", err)
	}
}
