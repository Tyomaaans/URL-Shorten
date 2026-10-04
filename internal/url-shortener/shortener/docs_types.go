package shortener

type ShortenerResponseData struct {
	Shorten ShortenResponse `json:"shorten"`
}

type ShortenerSwaggerResponse struct {
	Success bool                  `json:"success"`
	Message string                `json:"message,omitempty"`
	Data    ShortenerResponseData `json:"data,omitempty"`
}

type ShortensResponseData struct {
	Shorten PaginatedShortenResponse `json:"shortens"`
}

type ShortensListSwaggerResponse struct {
	Success bool                 `json:"success"`
	Message string               `json:"message,omitempty"`
	Data    ShortensResponseData `json:"data,omitempty"`
}
