// Package models describe models.
package models

// ShortenRequest instance
type ShortenRequest struct {
	URL string `json:"url"`
}

// ShortenResponse instance
type ShortenResponse struct {
	Result string `json:"result"`
}

// URLResponse instance
type URLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// BatchShortenRequest instance
type BatchShortenRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// BatchShortenResponse instance
type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}
