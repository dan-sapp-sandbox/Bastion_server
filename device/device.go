package device

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/dan-sapp-sandbox/Bastion_server/changeLog"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Device struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	IsOn bool   `json:"isOn"`
	Room string `json:"room"`
}

var (
	db               *sql.DB
	upgrader         = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	clients          = make(map[*websocket.Conn]bool)
	clientsMutex     sync.Mutex
	broadcastChannel = make(chan []byte)
)

func Setup(database *sql.DB) {
	db = database
	go handleBroadcasts()
}

func WebSocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close()

	clientsMutex.Lock()
	clients[conn] = true
	clientsMutex.Unlock()

	log.Println("Client connected via WebSocket")

	devices, err := fetchAllDevices()
	if err != nil {
		log.Println("Error fetching devices for WebSocket:", err)
		return
	}

	initialMessage := map[string]interface{}{
		"action":  "init",
		"devices": devices,
	}

	if err := conn.WriteJSON(initialMessage); err != nil {
		log.Println("Error sending initial devices to WebSocket client:", err)
	}

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket client disconnected:", err)
			clientsMutex.Lock()
			delete(clients, conn)
			clientsMutex.Unlock()
			break
		}
	}
}

func fetchAllDevices() ([]Device, error) {
	devices := []Device{}

	query := "SELECT id, name, room, type, isOn FROM devices"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var device Device
		if err := rows.Scan(&device.ID, &device.Name, &device.Room, &device.Type, &device.IsOn); err != nil {
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

	result, err := db.Exec("INSERT INTO devices (name, room, type, isOn) VALUES (?, ?, ?, ?)", newDevice.Name, newDevice.Room, newDevice.Type, newDevice.IsOn)
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
	sendDeviceUpdate("add")

	changeDescription := "Added new " + newDevice.Type + ": " + newDevice.Name
	if err := changeLog.AddEntryToLog(changeDescription, "add"); err != nil {
		log.Printf("Failed to log change: %v", err)
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Resource created successfully.",
		"data":    newDevice,
	})
}

func EditDevice(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid device ID.",
			"error":   err.Error(),
		})
		return
	}

	var updatedDevice Device
	if err := c.BindJSON(&updatedDevice); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Error updating resource.",
			"error":   err.Error(),
		})
		return
	}

	_, err = db.Exec("UPDATE devices SET name = ?, room = ?, type = ?, isOn = ? WHERE id = ?",
		updatedDevice.Name,
		updatedDevice.Room,
		updatedDevice.Type,
		updatedDevice.IsOn,
		id,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Error updating resource.",
			"error":   err.Error(),
		})
		return
	}

	updatedDevice.ID = id
	sendDeviceUpdate("update")

	changeDescription := "Edited device: " + updatedDevice.Name
	if err := changeLog.AddEntryToLog(changeDescription, "edit"); err != nil {
		log.Printf("Failed to log change: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Resource updated successfully.",
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
	err = db.QueryRow("SELECT id, name, type FROM devices WHERE id = ?", id).Scan(&device.ID, &device.Name, &device.Type)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Device not found."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Error fetching device."})
		return
	}

	_, err = db.Exec("DELETE FROM devices WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Error deleting device."})
		return
	}

	sendDeviceUpdate("delete")

	changeDescription := fmt.Sprintf("Deleted device %s '%s'", device.Type, device.Name)
	if err := changeLog.AddEntryToLog(changeDescription, "delete"); err != nil {
		log.Printf("Failed to log change: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Device deleted successfully."})
}

func sendDeviceUpdate(action string) {
	devices, err := fetchAllDevices()
	if err != nil {
		log.Println("Error fetching full device list:", err)
		return
	}

	update := map[string]interface{}{
		"action":  action,
		"devices": devices,
	}

	data, err := json.Marshal(update)
	if err != nil {
		log.Println("Error marshaling device update:", err)
		return
	}

	broadcastChannel <- data
}

func handleBroadcasts() {
	for {
		message := <-broadcastChannel
		clientsMutex.Lock()
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Println("WebSocket error:", err)
				client.Close()
				delete(clients, client)
			}
		}
		clientsMutex.Unlock()
	}
}

// CreateTable creates the devices table if it does not exist
func CreateTable() {
	createTableSQL := `CREATE TABLE IF NOT EXISTS devices (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        type TEXT NOT NULL,
        isOn BOOLEAN NOT NULL DEFAULT 0,
        room TEXT NOT NULL
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
		insertSQL := `INSERT INTO devices (name, room, type, isOn) VALUES
			('Main', 'Living Room', 'light', false),
			('Side', 'Living Room', 'light', true),
			('Front Door', 'Living', 'lock', true),
			('Kitchen Table', 'Kitchen', 'light', false),
			('Stove', 'Kitchen', 'light', true),
			('Main', 'Kitchen', 'speaker', true),
			('Vanity', 'Bathroom', 'light', false),
			('Main', 'Bedroom', 'light', true),
			('Main', 'Bedroom', 'fan', true),
			('Back Door', 'Other', 'lock', true)`

		_, err := db.Exec(insertSQL)
		if err != nil {
			log.Fatal("Failed to seed data: ", err)
		}
		log.Println("Database seeded with default devices.")
	}
}
