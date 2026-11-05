package release

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	AssetAccept = "application/octet-stream"
	APKMime     = "application/vnd.android.package-archive"
)

var (
	ErrNoToken   = errors.New("release: GITHUB_TOKEN is not set, so the APK cannot be fetched")
	ErrAssetGone = errors.New("release: the release has no APK for that version")
)

type Stream struct {
	Body   io.ReadCloser
	Length int64
	Status int
}

func stripAuth(request *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("release: too many redirects")
	}
	request.Header.Del("Authorization")
	request.Header.Del("X-GitHub-Api-Version")
	return nil
}

func (l *Lookup) assetClient() *http.Client {
	base := *l.client
	base.CheckRedirect = stripAuth
	return &base
}

func (l *Lookup) AssetFor(ctx context.Context, version string) (Details, error) {
	details, err := l.Latest(ctx)
	if err != nil {
		return Details{}, err
	}
	wanted := Version(version)
	if wanted == "" || wanted == details.Version {
		return details, nil
	}
	found, err := l.findVersion(ctx, wanted)
	if err != nil {
		return Details{}, err
	}
	return found, nil
}

func (l *Lookup) findVersion(ctx context.Context, version string) (Details, error) {
	releases, err := l.list(ctx)
	if err != nil {
		return Details{}, err
	}
	for _, entry := range releases {
		tag := Version(entry.TagName)
		if tag == "" {
			tag = Version(entry.Name)
		}
		if tag != version {
			continue
		}
		asset := assetOf(entry.Assets, tag)
		if asset.APIURL == "" {
			return Details{}, ErrAssetGone
		}
		return Details{
			Version:     tag,
			VersionCode: VersionCode(tag),
			APKURL:      asset.URL,
			AssetAPI:    asset.APIURL,
			Size:        asset.Size,
			Notes:       strings.TrimSpace(entry.Body),
			Required:    IsRequired(entry.Body),
			Downloads:   assetDownloads(entry.Assets),
		}, nil
	}
	return Details{}, ErrAssetGone
}

func (l *Lookup) Open(ctx context.Context, assetAPI string) (Stream, error) {
	if strings.TrimSpace(l.token) == "" {
		return Stream{}, ErrNoToken
	}
	if strings.TrimSpace(assetAPI) == "" {
		return Stream{}, ErrAssetGone
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, assetAPI, nil)
	if err != nil {
		return Stream{}, fmt.Errorf("release: could not build the asset request: %w", err)
	}
	request.Header.Set("Accept", AssetAccept)
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Authorization", "Bearer "+l.token)
	response, err := l.assetClient().Do(request)
	if err != nil {
		return Stream{}, fmt.Errorf("release: could not reach GitHub for the APK: %w", err)
	}
	if response.StatusCode >= http.StatusBadRequest {
		status := response.StatusCode
		_ = response.Body.Close()
		if status == http.StatusNotFound {
			return Stream{Status: status}, ErrAssetGone
		}
		return Stream{Status: status}, fmt.Errorf("release: GitHub answered %d %s for the APK",
			status, http.StatusText(status))
	}
	return Stream{Body: response.Body, Length: response.ContentLength, Status: response.StatusCode}, nil
}
