package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/tiger-box/URL-Shortener/internal"
)

type URL struct {
	shortener *internal.URLShortener
	logger    *slog.Logger
}

func NewURL(shortener *internal.URLShortener, logger *slog.Logger) *URL {
	return &URL{shortener: shortener, logger: logger}
}

type shorterRequest struct {
	URL string `json:"url"`
}

type shorterResponse struct {
	ShortURL string `json:"short_url"`
}

func (h *URL) GetShorter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req shorterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad request")
		h.logger.WarnContext(ctx, "bad request", "err", err)
		return
	}

	url, err := parseURL(req.URL)
	if url == "" {
		writeError(w, http.StatusBadRequest, "invalid URL")
		h.logger.WarnContext(ctx, "bad request", "err", err)
		return
	}

	short, err := h.shortener.Create(url)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		h.logger.ErrorContext(ctx, "failded create short url", "err", err)
		return
	}

	writeJSON(w, http.StatusCreated, shorterResponse{ShortURL: short})
	h.logger.InfoContext(ctx, "short url created")
}

type urlResponse struct {
	URL string `json:"url"`
}

func (h *URL) GetURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	code := r.PathValue("code")

	if code == "" {
		writeError(w, http.StatusBadRequest, "short url is required")
		h.logger.WarnContext(ctx, "short url is empty")
		return
	}

	url, err := h.shortener.Get(code)
	if err != nil {
		writeError(w, http.StatusNotFound, "url not found")
		h.logger.ErrorContext(ctx, "error url", "err", err)
		return
	}

	writeJSON(w, http.StatusOK, urlResponse{URL: url})
}

func parseURL(rawURL string) (string, error) {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}

	if u.Scheme == "" || u.Host == "" {
		return "", errors.New("parse url: host or scheme is empty")
	}

	return u.String(), nil
}
