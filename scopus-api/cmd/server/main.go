package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"scopus-api/internal/config"
	"scopus-api/internal/handler"
	"scopus-api/internal/middleware"
)

func main() {

	godotenv.Load()
	config.InitDB()

	r := gin.Default()

	r.GET("/research", middleware.CheckPackage(), handler.GetResearch)
	r.GET("/analytics", middleware.CheckPackage(), handler.GetAnalytics)
	r.GET("/reset", handler.ResetUsage)
	r.GET("/export", middleware.CheckPackage(), handler.ExportCSV)

	fmt.Println("Server running at http://localhost:8080")
	r.Run(":8080")
}