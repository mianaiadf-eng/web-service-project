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

func (s *ScopusService) GetResearch(userID string, limit int) ([]model.Research, error) {

	seen, err := repository.GetUserHistoryDOI(userID)
	if err != nil {
		return nil, err
	}

	cached, err := repository.GetAllResearch()
	if err != nil {
		return nil, err
	}

	var filtered []model.Research

	for _, r := range cached {
		if r.DOI != nil && seen[*r.DOI] {
			continue
		}
		filtered = append(filtered, r)

		if len(filtered) >= limit {
			break
		}
	}

	if len(filtered) > 0 {
		repository.SaveUserHistory(userID, filtered)
		return filtered, nil
	}

	apiKey := os.Getenv("SCOPUS_API_KEY")

	client := http.Client{Timeout: 10 * time.Second}

	var results []model.Research

	query := "AFFILCOUNTRY(Thailand)"
	encoded := url.QueryEscape(query)

	fullURL := fmt.Sprintf(
		"https://api.elsevier.com/content/search/scopus?query=%s&count=%d&apiKey=%s",
		encoded, limit, apiKey,
	)

	resp, err := client.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&data)

	sr := data["search-results"].(map[string]interface{})
	entries := sr["entry"].([]interface{})

	for _, e := range entries {

		item := e.(map[string]interface{})

		title := getString(item, "dc:title")
		journal := getString(item, "prism:publicationName")

		year := 0
		if d, ok := item["prism:coverDate"].(string); ok {
			year, _ = strconv.Atoi(d[:4])
		}

		var doi *string
		if d, ok := item["prism:doi"].(string); ok {
			doi = &d
		}

		results = append(results, model.Research{
			Title:   title,
			Journal: journal,
			Year:    year,
			DOI:     doi,
		})
	}

	repository.SaveResearch(results)
	repository.SaveUserHistory(userID, results)

	return results, nil
}

func (s *ScopusService) GetResearchWithFilter(year, university string) ([]model.Research, error) {
	return repository.GetResearchWithFilter(year, university)
}