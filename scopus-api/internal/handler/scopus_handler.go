package handler

import (
	"net/http"

	"scopus-api/internal/model"
	"scopus-api/internal/service"

	"github.com/gin-gonic/gin"
)

func GetResearch(c *gin.Context) {

	userID := c.GetString("userID")
	pkg := c.GetString("package")

	year := c.Query("year")
	university := c.Query("university")

	// 🔥 ดึง limit จาก middleware
	limit := c.GetInt("dataLimit")

	s := service.NewScopusService()

	var (
		data []model.Research
		err  error
	)

	if year != "" || university != "" {
		data, err = s.GetResearchWithFilter(year, university)
	} else {
		data, err = s.GetResearch(userID, limit)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":    userID,
		"package": pkg,
		"count":   len(data),
		"data":    data,
	})
}