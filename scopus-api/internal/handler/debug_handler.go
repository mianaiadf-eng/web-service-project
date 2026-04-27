package handler

import (
	"net/http"

	"scopus-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func ResetUsage(c *gin.Context) {

	middleware.ResetUsage() // เรียกไปเคลียร์

	c.JSON(http.StatusOK, gin.H{
		"message": "usage reset success",
	})
}