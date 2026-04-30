package middleware

import (
	"net/http"
	"sync"
	"time"

	"scopus-api/internal/repository"

	"github.com/gin-gonic/gin"
)

var usage = map[string]int{}
var mu sync.Mutex
var lastReset = time.Now().Day()

func CheckPackage() gin.HandlerFunc {
	return func(c *gin.Context) {

		apiKey := c.GetHeader("x-api-key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing api key"})
			c.Abort()
			return
		}

		userID, pkg, err := repository.GetUserByAPIKey(apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			c.Abort()
			return
		}

		mu.Lock()

		// 🔥 reset ทุกวัน
		if time.Now().Day() != lastReset {
			usage = map[string]int{}
			lastReset = time.Now().Day()
		}

		path := c.FullPath()

		skip := map[string]bool{
			"/register": true,
			"/login":    true,
			"/upgrade":  true,
			"/limit":    true,
		}

		// ✅ นับเฉพาะ route ที่ไม่อยู่ใน skip
		if !skip[path] {
			usage[userID]++
		}

		var dailyLimit int
		var dataLimit int
		var delay time.Duration

		switch pkg {

		case "free":
			dailyLimit = 10
			dataLimit = 10
			delay = 1 * time.Second

			if c.Query("year") != "" || c.Query("university") != "" {
				mu.Unlock()
				c.JSON(http.StatusForbidden, gin.H{
					"error": "free package: filter not allowed",
				})
				c.Abort()
				return
			}

		case "basic":
			dailyLimit = 50
			dataLimit = 100

		case "pro":
			dailyLimit = 200
			dataLimit = 500

		default:
			mu.Unlock()
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid package"})
			c.Abort()
			return
		}

		// 🔥 เช็ค limit ต่อวัน
		if usage[userID] > dailyLimit {
			mu.Unlock()

			now := time.Now()
			nextReset := time.Date(
				now.Year(), now.Month(), now.Day()+1,
				0, 0, 0, 0, now.Location(),
			)

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":      "daily limit reached",
				"next_reset": nextReset.Format("2006-01-02 15:04"),
			})
			c.Abort()
			return
		}

		mu.Unlock()

		// 🔥 ส่งค่าไป handler
		c.Set("userID", userID)
		c.Set("package", pkg)
		c.Set("dataLimit", dataLimit)

		//ฟาเซีย แก้ให้มีเส้น /limit
		c.Set("dailyLimit", dailyLimit)
		c.Set("usage", usage[userID])

		if delay > 0 {
			time.Sleep(delay)
		}

		c.Next()
	}
}
func ResetUsage() {
	mu.Lock()
	defer mu.Unlock()

	usage = map[string]int{}
}

func ResetUserUsage(userID string) {
	mu.Lock()
	defer mu.Unlock()

	delete(usage, userID)
}