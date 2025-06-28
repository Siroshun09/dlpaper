package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/Sirohun09/dlpaper/libs"
)

const (
	baseUrl        = "{api-server}/v3/projects/{project-name}/versions/{project-version}"
	latestBuildUrl = baseUrl + "/builds/latest"
)

type Client interface {
	GetLatestBuild(ctx context.Context) (BuildResponse, error)
	DownloadFile(ctx context.Context, url string, writer io.Writer) error
}

func NewClient(server string) Client {
	return client{server}
}

type client struct {
	server string
}

func (c client) GetLatestBuild(ctx context.Context) (BuildResponse, error) {
	url, err := libs.FormatString(ctx, latestBuildUrl, libs.ApiServerKey, libs.ProjectNameKey, libs.ProjectVersionKey)
	if err != nil {
		return BuildResponse{}, err
	}

	var response BuildResponse
	if err := getAndDecode(ctx, url, &response); err != nil {
		return BuildResponse{}, err
	}

	return response, nil
}

func (c client) DownloadFile(ctx context.Context, url string, writer io.Writer) error {
	return get(ctx, url, func(body io.ReadCloser) error {
		if _, err := io.Copy(writer, body); err != nil {
			return err
		}
		return nil
	})
}

func get(ctx context.Context, url string, bodyProcessor func(body io.ReadCloser) error) (returnErr error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "dlpaper (compatible; +https://github.com/Siroshun09/dlpaper)")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New(resp.Status)
	}

	body := resp.Body

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			returnErr = errors.Join(returnErr, err)
		}
	}(body)

	if err = bodyProcessor(body); err != nil {
		return err
	}

	return nil
}

func getAndDecode[T any](ctx context.Context, url string, response *T) error {
	return get(ctx, url, func(body io.ReadCloser) error {
		if err := json.NewDecoder(body).Decode(response); err != nil {
			return err
		}
		return nil
	})
}
