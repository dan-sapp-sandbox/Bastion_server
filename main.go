package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/dan-sapp-sandbox/Bastion_server/changeLog"
	"github.com/dan-sapp-sandbox/Bastion_server/device"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

func main() {
	devicesDB, err := sql.Open("sqlite", "./devices.db")
	if err != nil {
		log.Fatal("Error opening devices database:", err)
	}
	defer devicesDB.Close()

	device.Setup(devicesDB)
	device.CreateTable()

	logDB, err := sql.Open("sqlite", "./db_log.db")
	if err != nil {
		log.Fatal("Error opening log database:", err)
	}
	defer logDB.Close()

	changeLog.Setup(logDB)
	changeLog.CreateTable()

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},                                                      // Allow all origins for now
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},                // Allow required methods
		AllowHeaders:     []string{"Content-Type", "Authorization", "Upgrade", "Connection"}, // Allowed headers
		AllowCredentials: true,
	}))

	// Explicit OPTIONS handling (required for preflight requests)
	r.OPTIONS("/devices/:id", func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Status(http.StatusOK)
	})

	r.GET("/devices/ws", device.WebSocketHandler)

	r.POST("/add-device", device.AddDevice)
	r.PUT("/edit-device/:id", device.EditDevice)
	r.DELETE("/delete-device/:id", device.DeleteDevice)

	r.GET("/change-log", changeLog.GetChangeLog)

	r.Run()
}
