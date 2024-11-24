package device

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/dan-sapp-sandbox/Bastion_server/changeLog"
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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("perPage", "100"))

	if page < 1 {
		page = 1
	}

	offset := (page - 1) * perPage
	devices := []Device{}

	query := "SELECT id, name, type, isOn FROM devices LIMIT ? OFFSET ?"
	rows, err := db.Query(query, perPage, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error retrieving resources.",
			"error":   err.Error(),
		})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var u Device
		if err := rows.Scan(&u.ID, &u.Name, &u.Type, &u.IsOn); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "Error retrieving resources.",
				"error":   err.Error(),
			})
			return
		}
		devices = append(devices, u)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Resources retrieved successfully.",
		"data":    devices,
	})
}

func fetchDevices() ([]Device, error) {
	query := "SELECT id, name, type, isOn FROM devices"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var device Device
		if err := rows.Scan(&device.ID, &device.Name, &device.Type, &device.IsOn); err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}
	return devices, nil
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

	changeDescription := "Added new " + newDevice.Type + ": " + newDevice.Name
	if err := changeLog.AddEntryToLog(changeDescription); err != nil {
		log.Printf("Failed to log change: %v", err)
	}

	devices, err := fetchDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error retrieving resources.",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Resource created successfully.",
		"data":    devices,
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

	devices, err := fetchDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error retrieving resources.",
			"error":   err.Error(),
		})
		return
	}
	var matchingDevice *Device
	for _, device := range devices {
		if device.ID == id {
			matchingDevice = &device
			break
		}
	}

	changeDescription := "Edited " + matchingDevice.Name + ": Updated name to '" + updatedDevice.Name + "'"
	if err := changeLog.AddEntryToLog(changeDescription); err != nil {
		log.Printf("Failed to log change: %v", err)
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Resource updated successfully.",
		"data":    devices,
	})
}

func DeleteDevice(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid device ID.",
			"error":   err.Error(),
		})
		return
	}

	var device Device
	err = db.QueryRow("SELECT id, type, name FROM devices WHERE id = ?", id).Scan(&device.ID, &device.Type, &device.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Device not found.",
				"error":   "Device with given ID not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error fetching device.",
			"error":   err.Error(),
		})
		return
	}

	result, err := db.Exec("DELETE FROM devices WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error deleting device.",
			"error":   err.Error(),
		})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Device not found.",
			"error":   "No device was deleted",
		})
		return
	}

	changeDescription := fmt.Sprintf("Deleted %s '%s'", device.Type, device.Name)
	if err := changeLog.AddEntryToLog(changeDescription); err != nil {
		log.Printf("Failed to log change: %v", err)
	}

	devices, err := fetchDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error retrieving updated devices list.",
			"error":   err.Error(),
		})
		return
	}

	filteredDevices := []Device{}
	for _, dev := range devices {
		if dev.ID != id {
			filteredDevices = append(filteredDevices, dev)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Device deleted successfully.",
		"data":    filteredDevices,
	})
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

	seedData()
}

func seedData() {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&count)
	if err != nil {
		log.Fatal("Failed to check device count: ", err)
	}

	if count == 0 {
		insertSQL := `INSERT INTO devices (name, type, isOn) VALUES
			('Living Room', 'light', true),
			('Kitchen', 'light', false),
			('Bathroom', 'light', true),
			('Bedroom', 'fan', false),
			('Front Door', 'lock', true),
			('Back Door', 'lock', false)`

		_, err := db.Exec(insertSQL)
		if err != nil {
			log.Fatal("Failed to seed data: ", err)
		}
		log.Println("Database seeded with default devices.")
	}
}
