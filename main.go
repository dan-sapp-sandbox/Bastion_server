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
	db, err := sql.Open("sqlite", "./devices.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	device.Setup(db)
	device.CreateTable()

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8081", "https://lights-iota.vercel.app"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.GET("/devices", device.ListDevices)
	r.POST("/devices", device.AddDevice)
	r.PUT("/devices/:id", device.EditDevice)
	r.DELETE("/devices/:id", device.DeleteDevice)

	r.Run()
}
