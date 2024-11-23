package main

import (
	"database/sql"
	"log"

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
		AllowOrigins:     []string{"http://localhost:8081", "https://lights-iota.vercel.app"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.GET("/devices", device.ListDevices)
	r.POST("/devices", device.AddDevice)
	r.PUT("/devices/:id", device.EditDevice)
	r.DELETE("/devices/:id", device.DeleteDevice)

	r.GET("/changeLog", changeLog.GetChangeLog)

	r.Run()
}
