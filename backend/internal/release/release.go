package release

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultRepo     = "raHI-79/250-project"
	DefaultAPI      = "https://api.github.com"
	CacheTTL        = 10 * time.Minute
	RequiredMarker  = "[required]"
	Timeout         = 10 * time.Second
	assetPrefix     = "sust-cse-portal-"
	assetSuffix     = ".apk"
	maxBody         = 1 << 20
	versionCodeStep = 100
)

var ErrNoAsset = errors.New("release: the newest release has no APK asset")

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
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Body    string  `json:"body"`
	Draft   bool    `json:"draft"`
	Assets  []asset `json:"assets"`
}

type Lookup struct {
	repo    string
	api     string
	client  *http.Client
	now     func() time.Time
	guard   sync.Mutex
	cached  Details
	freshAt time.Time
	failed  error
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

func (l *Lookup) URL() string {
	return fmt.Sprintf("%s/repos/%s/releases/latest", l.api, l.repo)
}

func (l *Lookup) Latest(ctx context.Context) (Details, error) {
	l.guard.Lock()
	defer l.guard.Unlock()
	now := l.now()
	if !l.freshAt.IsZero() && now.Sub(l.freshAt) < CacheTTL {
		return l.cached, l.failed
	}
	details, err := l.fetch(ctx)
	if err != nil && !errors.Is(err, ErrNoAsset) && !l.freshAt.IsZero() && l.failed == nil {
		return l.cached, nil
	}
	l.cached = details
	l.failed = err
	l.freshAt = now
	return details, err
}

func (l *Lookup) fetch(ctx context.Context) (Details, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, l.URL(), nil)
	if err != nil {
		return Details{}, fmt.Errorf("release: could not build the request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	response, err := l.client.Do(request)
	if err != nil {
		return Details{}, fmt.Errorf("release: could not reach GitHub: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode >= http.StatusBadRequest {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxBody))
		return Details{}, fmt.Errorf("release: GitHub answered with status %d", response.StatusCode)
	}
	var latest payload
	if err := json.NewDecoder(io.LimitReader(response.Body, maxBody)).Decode(&latest); err != nil {
		return Details{}, fmt.Errorf("release: GitHub sent something unreadable: %w", err)
	}
	version := Version(latest.TagName)
	if version == "" {
		version = Version(latest.Name)
	}
	details := Details{
		Version:     version,
		VersionCode: VersionCode(version),
		Notes:       strings.TrimSpace(latest.Body),
		Required:    IsRequired(latest.Body),
	}
	details.APKURL = assetURL(latest.Assets, version)
	if details.APKURL == "" {
		return details, ErrNoAsset
	}
	return details, nil
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
	for _, item := range assets {
		if strings.HasPrefix(item.Name, assetPrefix) && strings.HasSuffix(item.Name, assetSuffix) &&
			item.URL != "" {
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
