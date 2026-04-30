package handler

import (

	"fmt"
	"encoding/csv"  
	"net/http"
	"strconv"  

	"time"   // 👈 เพิ่มบรรทัดนี้
	"scopus-api/internal/model"
	"scopus-api/internal/service"
	"scopus-api/internal/repository"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"
	"scopus-api/internal/middleware"
)

func GetResearch(c *gin.Context) {

	userID := c.GetString("userID")
	pkg := c.GetString("package")

	year := c.Query("year")
	university := c.Query("university")
	journal := c.Query("journal")

	limit := c.GetInt("dataLimit")
	analyticsReq := c.Query("analytics")

	s := service.NewScopusService()

	var (
		data []model.Research
		err  error
	)

	reqLimitStr := c.Query("limit")

if reqLimitStr != "" {
    reqLimit, err := strconv.Atoi(reqLimitStr)
    if err == nil && reqLimit > 0 {

        // ✅ ห้ามเกิน package limit
        if reqLimit < limit {
            limit = reqLimit
        }
    }
}
	// 👉 readable flags
	hasFilter := year != "" || university != "" || journal != ""
	wantAnalytics := analyticsReq == "true"

	// 🔒 FREE GUARD
	if pkg != "pro" && (hasFilter || wantAnalytics) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "filter & analytics available for pro package only",
		})
		return
	}

	// 📦 FETCH DATA
	if hasFilter {
		data, err = s.GetResearchWithFilter(year, university, journal)
	} else {
		data, err = s.GetResearch(userID, limit)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 📤 BASE RESPONSE
	response := gin.H{
		//"user":    userID,
		"package": pkg,
		"count":   len(data),
		"data":    data,
	}

	// 🧠 ADD ANALYTICS ONLY WHEN REQUESTED
	if wantAnalytics {
		response["analytics"] = s.AnalyzeResearch(data)
	}

	c.JSON(http.StatusOK, response)
}

func GetAnalytics(c *gin.Context) {

	pkg := c.GetString("package")
	byUniversity := c.Query("by_university") == "true"

	if pkg != "pro" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "analytics is pro feature only",
		})
		return
	}

	topJournals := c.Query("top_journals") == "true"
	byYear := c.Query("by_year") == "true"
	byJournal := c.Query("by_journal") == "true"

	s := service.NewScopusService()

	// ✅ ใช้ global data (ไม่เขียน history)
	data, err := repository.GetAllResearch()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	result := s.AnalyzeResearch(data)

	response := gin.H{}

	if topJournals {
		response["top_journals"] = result["top_journals"]
	}
	if byYear {
		response["by_year"] = result["by_year"]
	}
	if byJournal {
		response["by_journal"] = result["by_journal"]
	}
	if byUniversity {
		response["by_university"] = result["by_university"]
	}

	if len(response) == 0 {
		response = result
	}

	c.JSON(http.StatusOK, response)
}

func ExportCSV(c *gin.Context) {

	userID := c.GetString("userID")
	pkg := c.GetString("package")

	// 🔒 PRO ONLY
	if pkg != "pro" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "export CSV allowed for pro package only",
		})
		return
	}

	s := service.NewScopusService()

	data, err := s.ExportUserHistory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", "attachment; filename=history.csv")
	c.Header("Content-Type", "text/csv")

	writer := csv.NewWriter(c.Writer)

	// header
	writer.Write([]string{"Title", "Journal", "Year", "DOI", "University"})

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
			r.University,
		})
	}

	writer.Flush()
}

func GetLimit(c *gin.Context) {

	userID := c.GetString("userID")
	pkg := c.GetString("package")
	usage := c.GetInt("usage")
	dailyLimit := c.GetInt("dailyLimit")
	

	remaining := dailyLimit - usage
	if remaining < 0 {
		remaining = 0
	}

	now := time.Now()
	nextReset := time.Date(
		now.Year(), now.Month(), now.Day()+1,
		0, 0, 0, 0, now.Location(),
	)

	c.JSON(http.StatusOK, gin.H{
		"user":        userID,
		"package":     pkg,
		"daily_limit": dailyLimit,
		"used":        usage,
		"remaining":   remaining,
		"next_reset":  nextReset.Format("2006-01-02 15:04"),
	})
}

func Register(c *gin.Context) {

	var req struct {
		UserID   string `json:"user_id"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid input"})
		return
	}

	// ❗ เช็คก่อน (สำคัญ)
	if req.UserID == "" || req.Password == "" {
		c.JSON(400, gin.H{"error": "user_id and password required"})
		return
	}

	// 🔒 hash password จาก user
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		c.JSON(500, gin.H{"error": "hash error"})
		return
	}

	apiKey := uuid.New().String()

	err = repository.CreateUser(req.UserID, string(hashed), apiKey)
	if err != nil {
		c.JSON(400, gin.H{"error": "user already exists"})
		return
	}

	c.JSON(201, gin.H{
		"message": "register success",
		"data": gin.H{
			"api_key": apiKey,
			"package": "free",
		},
	})
}

func Login(c *gin.Context) {

	var req struct {
		UserID   string `json:"user_id"`
		Password string `json:"password"`
	}

	c.ShouldBindJSON(&req)

	user, err := repository.GetUserByUserID(req.UserID)
	if err != nil {
		c.JSON(401, gin.H{"error": "user not found"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		c.JSON(401, gin.H{"error": "wrong password"})
		return
	}

	c.JSON(200, gin.H{
		"api_key": user.APIKey,
		"package": user.Package,
	})
}

func UpgradePackage(c *gin.Context) {

	userID := c.GetString("userID")

	if userID == "" {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Package string `json:"package"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid input"})
		return
	}

	// validate package
	if req.Package != "free" && req.Package != "basic" && req.Package != "pro" {
		c.JSON(400, gin.H{"error": "invalid package"})
		return
	}

	// 🔥 debug (ช่วยหาปัญหา)
	fmt.Println("userID:", userID)
	fmt.Println("package:", req.Package)

	err := repository.UpdateUserPackage(userID, req.Package)
	if err != nil {
		fmt.Println("update error:", err) // 👈 ดู error จริง
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// reset usage เฉพาะ user
	middleware.ResetUserUsage(userID)

	c.JSON(200, gin.H{
		"message": "package updated",
		"package": req.Package,
	})
}