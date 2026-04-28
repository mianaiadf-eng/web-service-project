package repository

import (
	"context"
	"fmt"
	"scopus-api/internal/config"
	"scopus-api/internal/model"
)

func SaveResearch(results []model.Research) {

	ctx := context.Background()

	for _, r := range results {

		var doi interface{}
		if r.DOI != nil && *r.DOI != "" {
			doi = *r.DOI
		} else {
			doi = nil
		}

		_, err := config.DB.ExecContext(ctx, `
			INSERT INTO research (title, journal, year, doi, cited, university)
			VALUES ($1,$2,$3,$4,$5,$6)
			ON CONFLICT (doi) DO NOTHING
		`,
			r.Title,
			r.Journal,
			r.Year,
			doi,
			r.Cited,
			r.University,
		)

		if err != nil {
			fmt.Printf("Insert error: %v | title: %s\n", err, r.Title)
		}
	}
}

func GetAllResearch() ([]model.Research, error) {

	rows, err := config.DB.Query(`
		SELECT title, journal, year, doi, cited, university
		FROM research
		ORDER BY year DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.Research

	for rows.Next() {
		var r model.Research
		var doi *string

		err := rows.Scan(
			&r.Title,
			&r.Journal,
			&r.Year,
			&doi,
			&r.Cited,
			&r.University,
		)
		if err != nil {
			continue
		}

		r.DOI = doi
		results = append(results, r)
	}

	return results, nil
}

func GetResearchWithFilter(year string, university string) ([]model.Research, error) {

	query := `
		SELECT title, journal, year, doi, cited, university
		FROM research
		WHERE 1=1
	`

	args := []interface{}{}
	i := 1

	if year != "" {
		query += fmt.Sprintf(" AND year = $%d", i)
		args = append(args, year)
		i++
	}

	if university != "" {
		query += fmt.Sprintf(" AND university ILIKE $%d", i)
		args = append(args, "%"+university+"%")
		i++
	}

	query += " ORDER BY year DESC LIMIT 25"

	rows, err := config.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.Research

	for rows.Next() {
		var r model.Research
		var doi *string

		err := rows.Scan(
			&r.Title,
			&r.Journal,
			&r.Year,
			&doi,
			&r.Cited,
			&r.University,
		)
		if err != nil {
			continue
		}

		r.DOI = doi
		results = append(results, r)
	}

	return results, nil
}

func GetUserByAPIKey(apiKey string) (string, string, error) {
	var userID, pkg string

	err := config.DB.QueryRow(`
		SELECT user_id, package
		FROM users
		WHERE api_key = $1
	`, apiKey).Scan(&userID, &pkg)

	return userID, pkg, err
}

func GetUserHistoryDOI(userID string) (map[string]bool, error) {
	rows, err := config.DB.Query(`
		SELECT doi FROM user_research_history
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := make(map[string]bool)

	for rows.Next() {
		var doi string
		if err := rows.Scan(&doi); err == nil {
			seen[doi] = true
		}
	}

	return seen, nil
}

func SaveUserHistory(userID string, results []model.Research) error {

	for _, r := range results {
		if r.DOI == nil {
			continue
		}

		_, err := config.DB.Exec(`
			INSERT INTO user_research_history (user_id, doi, title, year)
			VALUES ($1,$2,$3,$4)
			ON CONFLICT DO NOTHING
		`, userID, *r.DOI, r.Title, r.Year)

		if err != nil {
			return err
		}
	}

	return nil
}