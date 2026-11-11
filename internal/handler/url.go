package handler

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/tiger-box/URL-Shortener/internal"
)

type URL struct {
	shortener *internal.URLShortener
}

func NewURL(shortener *internal.URLShortener) *URL {
	return &URL{shortener: shortener}
}

type shorterRequest struct {
	URL string `json:"url"`
}

type shorterResponse struct {
	ShortURL string `json:"short_url"`
}

func (h *URL) GetShorter(w http.ResponseWriter, r *http.Request) {
	var req shorterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad request")
		return
	}

	url := parseURL(req.URL)
	if url == "" {
		writeError(w, http.StatusBadRequest, "invalid URL")
		return
	}

	short, err := h.shortener.Create(url)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, shorterResponse{ShortURL: short})
}

type urlResponse struct {
	URL string `json:"url"`
}

func (h *URL) GetURL(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	if code == "" {
		writeError(w, http.StatusBadRequest, "short url is required")
		return
	}

	url, err := h.shortener.Get(code)
	if err != nil {
		writeError(w, http.StatusNotFound, "url not found")
		return
	}

	writeJSON(w, http.StatusOK, urlResponse{URL: url})
}

func parseURL(rawURL string) string {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return ""
	}

	if u.Scheme == "" || u.Host == "" {
		return ""
	}

	return u.String()
}
