package psn

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Service struct {
	client *http.Client
	logger *slog.Logger
}

type Result struct {
	Public    bool
	AvatarURL string
}

var avatarURLPattern = regexp.MustCompile(`https?://static-resource\.np\.community\.playstation\.net/avatar_m/[^"\\<]+`)

func New(logger *slog.Logger, timeout time.Duration) *Service {
	return &Service{
		client: &http.Client{Timeout: timeout},
		logger: logger,
	}
}

func (s *Service) Check(ctx context.Context, profileURL string) Result {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, profileURL, nil)
	if err != nil {
		s.logger.Warn("profile request build failed", "error", err)
		return Result{Public: false}
	}
	req.Header.Set("User-Agent", "psn-add/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Warn("profile fetch failed", "url", profileURL, "error", err)
		return Result{Public: false}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		s.logger.Warn("profile read failed", "url", profileURL, "error", err)
		return Result{Public: false}
	}

	html := string(body)
	public := resp.StatusCode >= 200 &&
		resp.StatusCode < 400 &&
		!strings.Contains(html, "error_search.png") &&
		!strings.Contains(html, "Something went wrong")
	return Result{
		Public:    public,
		AvatarURL: extractAvatarURL(html, public),
	}
}

func extractAvatarURL(html string, public bool) string {
	if !public {
		return ""
	}
	avatarURL := avatarURLPattern.FindString(html)
	if avatarURL == "" {
		return ""
	}
	return strings.Replace(avatarURL, "http://", "https://", 1)
}
