package service


import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"scopus-api/internal/model"
	"scopus-api/internal/repository"
)

type ScopusService struct{}

func NewScopusService() *ScopusService {
	return &ScopusService{}
}

// ------------------ helper ------------------
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return strings.TrimSpace(fmt.Sprint(v))
	}
	return ""
}

// ------------------ ดึงมหาลัย (เฉพาะไทย + university) ------------------
func (s *ScopusService) getUniversity(eid string) string {

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
		return ""
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ""
	}

	root, ok := data["abstracts-retrieval-response"].(map[string]interface{})
	if !ok {
		return ""
	}

	if affs, ok := root["affiliation"].([]interface{}); ok {

		for _, a := range affs {
			affMap, ok := a.(map[string]interface{})
			if !ok {
				continue
			}

			name := getString(affMap, "affilname")
			country := strings.ToLower(getString(affMap, "affiliation-country"))

			// 🔥 เฉพาะไทย + ต้องเป็นมหาลัย
			if strings.Contains(country, "thailand") &&
				strings.Contains(strings.ToLower(name), "university") {

				return name
			}
		}
	}

	return ""
}

// ------------------ main logic ------------------
func (s *ScopusService) GetResearch() ([]model.Research, error) {

	apiKey := os.Getenv("SCOPUS_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("missing SCOPUS_API_KEY")
	}

	// 🔥 ดึงเฉพาะประเทศไทย
	query := "AFFILCOUNTRY(Thailand)"
	encoded := url.QueryEscape(query)

	fullURL := fmt.Sprintf(
		"https://api.elsevier.com/content/search/scopus?query=%s&count=10&apiKey=%s",
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
		return []model.Research{}, nil
	}

	entriesRaw := sr["entry"]

	var entries []interface{}
	switch v := entriesRaw.(type) {
	case []interface{}:
		entries = v
	case map[string]interface{}:
		entries = []interface{}{v}
	default:
		return []model.Research{}, nil
	}

	var results []model.Research

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

		rawID := fmt.Sprint(item["dc:identifier"])
		eid := strings.TrimPrefix(rawID, "SCOPUS_ID:")

		var doi *string
		if d, ok := item["prism:doi"].(string); ok {
			doi = &d
		}

		// 🔥 ดึงมหาลัยไทยจริงเท่านั้น
		university := s.getUniversity(eid)

		// ❌ ตัดที่ไม่ใช่ไทยออก
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

	// 🔥 save ลง database
	repository.SaveResearch(results)

	return results, nil
}

func (s *ScopusService) GetResearchFromDB() ([]model.Research, error) {
	return repository.GetAllResearch()
}