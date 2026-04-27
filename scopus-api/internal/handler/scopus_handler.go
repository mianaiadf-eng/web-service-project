package handler

import (
	"net/http"

	"scopus-api/internal/model"
	"scopus-api/internal/service"

	"github.com/gin-gonic/gin"
)

func GetResearch(c *gin.Context) {

	year := c.Query("year")
	university := c.Query("university")

	// 🔥 รับ limit จาก middleware
	limitAny, exists := c.Get("dataLimit")
	limit := 25 // default กันพัง

	if exists {
		if l, ok := limitAny.(int); ok {
			limit = l
		}
	}

	s := service.NewScopusService()

	var (
		data []model.Research
		err  error
	)

	if year != "" || university != "" {
		data, err = s.GetResearchWithFilter(year, university, limit)
	} else {
		data, err = s.GetResearchWithLimit(limit)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(data),
		"data":  data,
	})
}