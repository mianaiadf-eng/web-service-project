package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var usage = map[string]int{}
var mu sync.Mutex
var lastReset = time.Now().Day()

func CheckPackage() gin.HandlerFunc {
	return func(c *gin.Context) {

		key := c.GetHeader("x-api-key")

		if key == "" {
			c.JSON(401, gin.H{"error": "missing api key"})
			c.Abort()
			return
		}

		path := c.Request.URL.Path

		// 🔥 lock กัน race condition
		mu.Lock()

		// 🔥 reset usage ทุกวัน
		if time.Now().Day() != lastReset {
			usage = map[string]int{}
			lastReset = time.Now().Day()
		}

		usage[key]++

		var limit int
		var dataLimit int
		var delay time.Duration

		switch key {

		// 🥇 FREE
		case "free":
			if path != "/research" {
				mu.Unlock()
				c.JSON(403, gin.H{"error": "free package: access denied"})
				c.Abort()
				return
			}

			// ❗ กัน filter
			if c.Query("year") != "" || c.Query("university") != "" {
				mu.Unlock()
				c.JSON(403, gin.H{
					"error": "free package: filter not allowed",
				})
				c.Abort()
				return
			}

			limit = 10
			dataLimit = 20
			delay = 1 * time.Second

		// 🥈 BASIC
		case "basic":
			if path != "/research" {
				mu.Unlock()
				c.JSON(403, gin.H{"error": "basic package: access denied"})
				c.Abort()
				return
			}

			limit = 20
			dataLimit = 25

		// 🥇🥇 PRO
		case "pro":
			limit = 100
			dataLimit = 100

		default:
			mu.Unlock()
			c.JSON(403, gin.H{"error": "invalid api key"})
			c.Abort()
			return
		}

		// 🔥 เช็ค limit
		if usage[key] > limit {
			mu.Unlock()

			now := time.Now()
			nextReset := time.Date(
				now.Year(), now.Month(), now.Day()+1,
				0, 0, 0, 0, now.Location(),
			).Format("2006-01-02 15:04")

			c.JSON(429, gin.H{
				"error":       "daily limit reached",
				"next_reset":  nextReset,
				"suggestion":  "upgrade to basic/pro",
			})

			c.Abort()
			return
		}

		mu.Unlock()

		// 👉 ส่งค่าไป handler
		c.Set("dataLimit", dataLimit)
		c.Set("package", key)

		// 👉 delay สำหรับ free
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