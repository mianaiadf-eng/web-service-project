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
	r.GET("/reset", handler.ResetUsage)

	fmt.Println("Server running at http://localhost:8080")
	r.Run(":8080")
}