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