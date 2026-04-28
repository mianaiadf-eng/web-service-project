package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"

	"scopus-api/internal/model"
	"scopus-api/internal/service"

	"github.com/gin-gonic/gin"
)

func GetResearch(c *gin.Context) {

	userID := c.GetString("userID")
	pkg := c.GetString("package")

	year := c.Query("year")
	university := c.Query("university")

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

/////////////////////////////////////////////////////
// 🔥 EXPORT CSV (แก้ใหม่ใช้ ExportAllResearch)
/////////////////////////////////////////////////////

func ExportCSV(c *gin.Context) {

	limit := c.GetInt("dataLimit")

	s := service.NewScopusService()

	data, err := s.ExportAllResearch(limit) // ✅ ใช้ function ใหม่
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename=research.csv")
	c.Header("Content-Type", "text/csv")

	writer := csv.NewWriter(c.Writer)

	// header
	writer.Write([]string{"Title", "Journal", "Year", "DOI", "Cited"})

	for _, r := range data {

		doi := ""
		if r.DOI != nil {
			doi = *r.DOI
		}

		writer.Write([]string{
			r.Title,
			r.Journal,
			strconv.Itoa(r.Year),
			doi,
			strconv.Itoa(r.Cited),
		})
	}

	writer.Flush()
}