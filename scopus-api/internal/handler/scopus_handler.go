package handler

import (
	"net/http"

	"scopus-api/internal/service"

	"github.com/gin-gonic/gin"
)

// ------------------ ดึงจาก Scopus ------------------
func GetScopus(c *gin.Context) {

	s := service.NewScopusService()

	data, err := s.GetResearch()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, data)
}

// ------------------ ดึงจาก Database ------------------
func GetResearch(c *gin.Context) {

	s := service.NewScopusService()

	data, err := s.GetResearchFromDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, data)
}