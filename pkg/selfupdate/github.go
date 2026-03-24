package selfupdate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

const (
	baseURL = "https://api.github.com"
)

type release struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
}

type releaseAsset struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// getLatestRelease returns the latest release for th given owner/repo
//
// https://docs.github.com/en/rest/reference/repos#get-the-latest-release
func getLatestRelease(owner, repo string) (*release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", baseURL, owner, repo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "new http req")
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "do http req")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("bad http code")
	}

	latest := release{}
	err = json.NewDecoder(resp.Body).Decode(&latest)
	if err != nil {
		return nil, errors.Wrap(err, "decode github response")
	}

	return &latest, nil
}

// getReleaseAsset downloads asset by it ID
//
// https://docs.github.com/en/rest/reference/repos#get-a-release-asset
func getReleaseAsset(owner, repo string, assetID int64) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/assets/%d", baseURL, owner, repo, assetID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "new http req")
	}
	req.Header.Add("Accept", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "do http req")
	}

	if resp.StatusCode != 200 {
		if err = resp.Body.Close(); err != nil {
			fmt.Println("resp.Body.Close()", err)
		}
		return nil, errors.New("bad http code")
	}

	return resp.Body, nil
}
