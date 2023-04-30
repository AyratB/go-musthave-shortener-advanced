// Package models describes main entities.
package models

// ShortenRequest describes request fields
type ShortenRequest struct {
	URL string `json:"url"`
}

// ShortenResponse describes response fields
type ShortenResponse struct {
	Result string `json:"result"`
}

// URLResponse describes URL response fields
type URLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// BatchShortenRequest describes request fields when we save batch
type BatchShortenRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// BatchShortenResponse describes response fields when we save batch
type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}
