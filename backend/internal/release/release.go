package release

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultRepo     = "raHI-79/250-project"
	DefaultAPI      = "https://api.github.com"
	DefaultWeb      = "https://github.com"
	CacheTTL        = 10 * time.Minute
	StaleTTL        = 6 * time.Hour
	RetryAfter      = time.Minute
	RequiredMarker  = "[required]"
	Timeout         = 10 * time.Second
	PageSize        = 30
	userAgent       = "sust-cse-portal"
	assetPrefix     = "sust-cse-portal-"
	assetSuffix     = ".apk"
	maxBody         = 1 << 20
	maxReason       = 200
	versionCodeStep = 100
)

var (
	ErrNoAsset     = errors.New("release: no published release carries an APK asset")
	ErrUnavailable = errors.New("release: GitHub could not be read and nothing is cached")
)

type Details struct {
	Version     string
	VersionCode int
	APKURL      string
	Notes       string
	Required    bool
}

type asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type payload struct {
	TagName    string  `json:"tag_name"`
	Name       string  `json:"name"`
	Body       string  `json:"body"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	Assets     []asset `json:"assets"`
}

type Lookup struct {
	repo    string
	api     string
	token   string
	client  *http.Client
	now     func() time.Time
	guard   sync.Mutex
	good    Details
	goodAt  time.Time
	noAsset bool
	triedAt time.Time
}

func New(repo, api string) *Lookup {
	return NewWith(repo, api, nil, nil)
}

func NewWith(repo, api string, client *http.Client, now func() time.Time) *Lookup {
	if strings.TrimSpace(repo) == "" {
		repo = DefaultRepo
	}
	if strings.TrimSpace(api) == "" {
		api = DefaultAPI
	}
	if client == nil {
		client = &http.Client{Timeout: Timeout}
	}
	if now == nil {
		now = time.Now
	}
	return &Lookup{repo: strings.Trim(repo, "/"), api: strings.TrimRight(api, "/"), client: client, now: now}
}

func (l *Lookup) WithToken(token string) *Lookup {
	l.token = strings.TrimSpace(token)
	return l
}

func (l *Lookup) URL() string {
	return fmt.Sprintf("%s/repos/%s/releases?per_page=%d", l.api, l.repo, PageSize)
}

func (l *Lookup) ReleasesPage() string {
	return fmt.Sprintf("%s/%s/releases/latest", DefaultWeb, l.repo)
}

func (l *Lookup) Latest(ctx context.Context) (Details, error) {
	l.guard.Lock()
	defer l.guard.Unlock()
	now := l.now()
	if !l.goodAt.IsZero() && now.Sub(l.goodAt) < CacheTTL {
		return l.answer()
	}
	if !l.triedAt.IsZero() && now.Sub(l.triedAt) < RetryAfter {
		return l.stale(now)
	}
	l.triedAt = now
	details, err := l.fetch(ctx)
	switch {
	case err == nil:
		l.good, l.goodAt, l.noAsset = details, now, false
		return details, nil
	case errors.Is(err, ErrNoAsset):
		l.good, l.goodAt, l.noAsset = Details{}, now, true
		return Details{}, ErrNoAsset
	}
	kept, missing := l.stale(now)
	if missing == nil {
		log.Printf("%v, serving the release cached %s ago",
			err, now.Sub(l.goodAt).Round(time.Second))
	} else {
		log.Printf("%v, nothing is cached", err)
	}
	return kept, missing
}

func (l *Lookup) answer() (Details, error) {
	if l.goodAt.IsZero() {
		return Details{}, ErrUnavailable
	}
	if l.noAsset {
		return Details{}, ErrNoAsset
	}
	return l.good, nil
}

func (l *Lookup) stale(now time.Time) (Details, error) {
	if l.goodAt.IsZero() || now.Sub(l.goodAt) >= StaleTTL {
		return Details{}, ErrUnavailable
	}
	return l.answer()
}

func (l *Lookup) fetch(ctx context.Context) (Details, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, l.URL(), nil)
	if err != nil {
		return Details{}, fmt.Errorf("release: could not build the request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", userAgent)
	if l.token != "" {
		request.Header.Set("Authorization", "Bearer "+l.token)
	}
	response, err := l.client.Do(request)
	if err != nil {
		return Details{}, fmt.Errorf("release: could not reach GitHub: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	body := io.LimitReader(response.Body, maxBody)
	if response.StatusCode >= http.StatusBadRequest {
		return Details{}, fmt.Errorf("release: GitHub answered %d %s (%s)",
			response.StatusCode, http.StatusText(response.StatusCode), reason(response, body))
	}
	var releases []payload
	if err := json.NewDecoder(body).Decode(&releases); err != nil {
		return Details{}, fmt.Errorf("release: GitHub sent something unreadable: %w", err)
	}
	for _, entry := range releases {
		if entry.Draft || entry.Prerelease {
			continue
		}
		version := Version(entry.TagName)
		if version == "" {
			version = Version(entry.Name)
		}
		url := assetURL(entry.Assets, version)
		if version == "" || url == "" {
			continue
		}
		return Details{
			Version:     version,
			VersionCode: VersionCode(version),
			APKURL:      url,
			Notes:       strings.TrimSpace(entry.Body),
			Required:    IsRequired(entry.Body),
		}, nil
	}
	return Details{}, ErrNoAsset
}

func reason(response *http.Response, body io.Reader) string {
	parts := make([]string, 0, 3)
	if left := response.Header.Get("X-RateLimit-Remaining"); left == "0" {
		parts = append(parts, "rate limit exhausted")
		if reset := response.Header.Get("X-RateLimit-Reset"); reset != "" {
			parts = append(parts, "resets at "+reset)
		}
	}
	if retry := response.Header.Get("Retry-After"); retry != "" {
		parts = append(parts, "retry after "+retry+"s")
	}
	var answer struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(body).Decode(&answer); err == nil {
		if text := strings.TrimSpace(answer.Message); text != "" {
			if len(text) > maxReason {
				text = text[:maxReason]
			}
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 {
		return "no reason given"
	}
	return strings.Join(parts, ", ")
}

func Version(tag string) string {
	return strings.TrimPrefix(strings.TrimSpace(tag), "v")
}

func IsRequired(notes string) bool {
	return strings.Contains(strings.ToLower(notes), RequiredMarker)
}

func AssetName(version string) string {
	return assetPrefix + version + assetSuffix
}

func assetURL(assets []asset, version string) string {
	wanted := AssetName(version)
	for _, item := range assets {
		if item.Name == wanted && item.URL != "" {
			return item.URL
		}
	}
	return ""
}

func VersionCode(version string) int {
	parts := strings.SplitN(Version(version), "-", 2)
	numbers := strings.Split(parts[0], ".")
	code := 0
	for index := 0; index < 3; index++ {
		piece := 0
		if index < len(numbers) {
			piece, _ = strconv.Atoi(strings.TrimSpace(numbers[index]))
		}
		if piece < 0 {
			piece = 0
		}
		code = code*versionCodeStep + piece
	}
	return code
}

func Newer(latest, installed string) bool {
	return VersionCode(latest) > VersionCode(installed)
}
