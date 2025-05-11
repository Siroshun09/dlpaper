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
	baseUrl     = "{api-server}/v2/projects/{project-name}/versions/{project-version}"
	buildsUrl   = baseUrl + "/builds"
	buildUrl    = baseUrl + "/builds/{build}"
	downloadUrl = buildUrl + "/downloads/{download-name}"
)

type Client interface {
	GetVersionBuilds(ctx context.Context) (BuildsResponse, error)
	Download(ctx context.Context, writer io.Writer) error
}

func NewClient(server string) Client {
	return client{server}
}

type client struct {
	server string
}

func (c client) GetVersionBuilds(ctx context.Context) (BuildsResponse, error) {
	url, err := libs.FormatString(ctx, buildsUrl, libs.ApiServerKey, libs.ProjectNameKey, libs.ProjectVersionKey)

	if err != nil {
		return BuildsResponse{}, err
	}

	var response BuildsResponse
	if err := getAndDecode(url, &response); err != nil {
		return BuildsResponse{}, err
	}

	return response, nil
}

func (c client) Download(ctx context.Context, writer io.Writer) error {
	url, err := libs.FormatString(ctx, downloadUrl, libs.ApiServerKey, libs.ProjectNameKey, libs.ProjectVersionKey, libs.BuildKey, libs.DownloadNameKey)

	if err != nil {
		return err
	}

	return get(url, func(body io.ReadCloser) error {
		if _, err := io.Copy(writer, body); err != nil {
			return err
		}
		return nil
	})
}

func get(url string, bodyProcessor func(body io.ReadCloser) error) (returnErr error) {
	resp, err := http.Get(url)

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

func getAndDecode[T any](url string, response *T) error {
	return get(url, func(body io.ReadCloser) error {
		if err := json.NewDecoder(body).Decode(response); err != nil {
			return err
		}
		return nil
	})
}
