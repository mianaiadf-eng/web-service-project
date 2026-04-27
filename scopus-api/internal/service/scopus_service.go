package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"scopus-api/internal/model"
	"scopus-api/internal/repository"
)

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

func (s *ScopusService) GetResearch() ([]model.Research, error) {

		// 🔥 1. เช็ค cache ก่อน
	cached, _ := repository.GetAllResearch()

	if len(cached) > 0 {
		fmt.Println("⚡ ใช้ข้อมูลจาก DB (cache)")
		return cached, nil
	}

	fmt.Println("🌐 ดึงจาก Scopus API")

	// 🔽 ของเดิมคุณต่อจากนี้ได้เลย

	apiKey := os.Getenv("SCOPUS_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("missing SCOPUS_API_KEY")
	}

	// 🔥 ดึงเฉพาะมหาลัยไทยตั้งแต่ต้น
	query := "AFFILCOUNTRY(Thailand) AND (AFFILORG(university) OR AFFILORG(univ))"
	encoded := url.QueryEscape(query)

	client := http.Client{Timeout: 10 * time.Second}

	var results []model.Research

	for start := 0; start < 2000; start += 25 {

		fullURL := fmt.Sprintf(
			"https://api.elsevier.com/content/search/scopus?query=%s&count=25&start=%d&apiKey=%s",
			encoded,
			start,
			apiKey,
		)

		resp, err := client.Get(fullURL)
		if err != nil {
			return nil, err
		}

		var data map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		sr, ok := data["search-results"].(map[string]interface{})
		if !ok {
			break
		}

		entriesRaw := sr["entry"]

		var entries []interface{}
		switch v := entriesRaw.(type) {
		case []interface{}:
			entries = v
		case map[string]interface{}:
			entries = []interface{}{v}
		default:
			break
		}

		if len(entries) == 0 {
			break
		}

		for _, e := range entries {

			item := e.(map[string]interface{})

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

			var doi *string
			if d, ok := item["prism:doi"].(string); ok {
				doi = &d
			}

			var university string

if affs, ok := item["affiliation"].([]interface{}); ok {

	for _, a := range affs {

		affMap, ok := a.(map[string]interface{})
		if !ok {
			continue
		}

		name := getString(affMap, "affilname")
		country := strings.ToLower(getString(affMap, "affiliation-country"))

		// ✅ เอาเฉพาะ "ไทย + มหาลัย"
		if strings.Contains(country, "thailand") &&
			(strings.Contains(strings.ToLower(name), "university") ||
				strings.Contains(strings.ToLower(name), "univ")) {

			university = name
			break
		}
	}
}

// ❌ ถ้าไม่ใช่ไทย → ไม่เอา
if university == "" {
	continue
}

			results = append(results, model.Research{
				Title:      title,
				Journal:    journal,
				Year:       year,
				DOI:        doi,
				Cited:      cited,
				Authors:    []string{},
				University: university,
			})
		}
	}

	repository.SaveResearch(results)

	return results, nil
}

func (s *ScopusService) GetResearchFromDB() ([]model.Research, error) {
	return repository.GetAllResearch()
}


func (s *ScopusService) GetResearchWithFilter(year, university string, limit int) ([]model.Research, error) {
	return repository.GetResearchWithFilter(year, university, limit)
}

func (s *ScopusService) GetResearchWithLimit(limit int) ([]model.Research, error) {
	return repository.GetResearchWithLimit(limit)
}