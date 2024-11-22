package main

import (
	"database/sql"
	"log"

	"github.com/dan-sapp-sandbox/Bastion_server/device"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

func main() {
	// Connect to SQLite Database
	db, err := sql.Open("sqlite", "./devices.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Setup device package and create table
	device.Setup(db)
	device.CreateTable()

	// Set up Gin
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8081", "https://lights-iota.vercel.app/"}, // Add your front-end origin
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Routes
	r.GET("/devices", device.ListDevices)
	r.POST("/devices", device.AddDevice)
	r.PUT("/devices/:id", device.EditDevice)
	r.DELETE("/devices/:id", device.DeleteDevice)

	// Start server
	r.Run() // listen and serve on 0.0.0.0:8080
}
