package main


import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"scopus-api/internal/config"
	"scopus-api/internal/handler"
	"scopus-api/internal/middleware"
)


type Research struct {
	Title   string   `json:"title"`
	Journal string   `json:"journal"`
	Year    int      `json:"year"`
	DOI     *string  `json:"doi"`
	Cited   int      `json:"cited"`
	Authors []string `json:"authors"`
	University string   `json:"university"`   // 🔥 เพิ่มตรงนี้
}

type ScopusService struct{}

func NewScopusService() *ScopusService {
	return &ScopusService{}
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return strings.TrimSpace(fmt.Sprint(v))
	}
	return ""
}

// ------------------ 🔥 LIST มหาลัยไทย ------------------
var thaiUniversities = []string{
	"Chulalongkorn University",
	"Mahidol University",
	"Chiang Mai University",
	"Kasetsart University",
	"Prince of Songkla University",
	"Thammasat University",
	"Khon Kaen University",
	"Silpakorn University",
	"King Mongkut's Institute of Technology Ladkrabang",
	"King Mongkut's University of Technology Thonburi",
	"King Mongkut's University of Technology North Bangkok",
	"Suranaree University of Technology",
	"Naresuan University",
	"Burapha University",
	"Mae Fah Luang University",
	"Walailak University",
	"Srinakharinwirot University",
}

// ------------------ 🔥 ดึง Authors ------------------
func (s *ScopusService) getAuthors(eid string) []string {
	apiKey := os.Getenv("SCOPUS_API_KEY")

	url := fmt.Sprintf(
		"https://api.elsevier.com/content/abstract/scopus_id/%s",
		eid,
	)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-ELS-APIKey", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return []string{}
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return []string{}
	}

	root, ok := data["abstracts-retrieval-response"].(map[string]interface{})
	if !ok {
		return []string{}
	}

	var authors []string

	if core, ok := root["coredata"].(map[string]interface{}); ok {
		if creator, ok := core["dc:creator"].(string); ok {
			authors = append(authors, creator)
		}
	}

	return authors
}

// ------------------ 🔥 ยิง Scopus ต่อ batch ------------------
func (s *ScopusService) fetchBatch(universities []string) ([]Research, error) {

	apiKey := os.Getenv("SCOPUS_API_KEY")

	var parts []string
	for _, u := range universities {
		parts = append(parts, fmt.Sprintf("AFFIL(%s)", u))
	}

	query := strings.Join(parts, " OR ")
	encoded := url.QueryEscape(query)

	fullURL := fmt.Sprintf(
		"https://api.elsevier.com/content/search/scopus?query=%s&apiKey=%s",
		encoded,
		apiKey,
	)

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	sr, ok := data["search-results"].(map[string]interface{})
	if !ok {
		return []Research{}, nil
	}

	entriesRaw := sr["entry"]

	var entries []interface{}
	switch v := entriesRaw.(type) {
	case []interface{}:
		entries = v
	case map[string]interface{}:
		entries = []interface{}{v}
	default:
		return []Research{}, nil
	}

	var results []Research

	for _, e := range entries {
		item := e.(map[string]interface{})
		university := ""

if affs, ok := item["affiliation"].([]interface{}); ok && len(affs) > 0 {
	if affMap, ok := affs[0].(map[string]interface{}); ok {
		university = getString(affMap, "affilname")
	}
}

		title := getString(item, "dc:title")
		journal := getString(item, "prism:publicationName")

		year := 0
		if d, ok := item["prism:coverDate"].(string); ok && len(d) >= 4 {
			year, _ = strconv.Atoi(d[:4])
		}

		cited := 0
		if c, ok := item["citedby-count"].(string); ok {
			cited, _ = strconv.Atoi(c)
		}



		rawID := fmt.Sprint(item["dc:identifier"])
		eid := strings.TrimPrefix(rawID, "SCOPUS_ID:")

		authors := s.getAuthors(eid)

		var doi *string
		if d, ok := item["prism:doi"].(string); ok {
			doi = &d
		}

		results = append(results, Research{
			Title:   title,
			Journal: journal,
			Year:    year,
			DOI:     doi,
			Cited:   cited,
			Authors: authors,
			University: university,
		})
	}

	return results, nil
}

// ------------------ 🔥 หลัก: batch + concurrency ------------------
func (s *ScopusService) GetResearch() ([]Research, error) {

	batchSize := 3
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allResults []Research

	for i := 0; i < len(thaiUniversities); i += batchSize {

		end := i + batchSize
		if end > len(thaiUniversities) {
			end = len(thaiUniversities)
		}

		batch := thaiUniversities[i:end]

		wg.Add(1)

		go func(b []string) {
			defer wg.Done()

			results, err := s.fetchBatch(b)
			if err != nil {
				return
			}

			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()
		}(batch)
	}

	wg.Wait()
	return allResults, nil
}

// ------------------ API ------------------
func getScopus(c *gin.Context) {
	service := NewScopusService()

	data, err := service.GetResearch()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, data)
}

func main() {

	// โหลด env
	godotenv.Load()

	// connect DB
	config.InitDB()

	r := gin.Default()

	//r.GET("/scopus", middleware.CheckPackage(), handler.GetScopus)
	r.GET("/research", middleware.CheckPackage(),handler.GetResearch)
	r.GET("/reset", handler.ResetUsage)

	fmt.Println("Server running at http://localhost:8080")

	r.Run(":8080")
}