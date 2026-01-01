package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/Sirohun09/dlpaper/api"
	"github.com/Sirohun09/dlpaper/libs"
)

func main() {
	err := libs.ParseFlags()

	if err != nil {
		slog.Error("An error found in flags.", slog.Any("error", err))
		os.Exit(1)
	}

	ctx := libs.CreateContext(libs.GetApiServer(), libs.GetProjectName(), libs.GetProjectVersion())

	client := api.NewClient(libs.GetApiServer())
	outputFile, err := libs.FormatString(ctx, libs.GetFilenameFormat(), libs.ProjectNameKey, libs.ProjectVersionKey)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to format output file.", slog.Any("error", err))
		os.Exit(1)
	}

	slog.InfoContext(ctx, "Checking updates...")

	latestBuildChan := make(chan result[api.BuildResponse], 1)
	defer close(latestBuildChan)

	lastUpdateTimeChan := make(chan result[*time.Time], 1)
	defer close(lastUpdateTimeChan)

	latestBuildCtx, latestBuildCancel := getLatestBuild(ctx, client, latestBuildChan)
	defer latestBuildCancel()

	getLastUpdateTime(outputFile, lastUpdateTimeChan)

	var latestBuild api.BuildResponse
	select {
	case r := <-latestBuildChan:
		if r.err != nil {
			slog.ErrorContext(ctx, "Failed to get builds.", slog.Any("error", r.err))
			os.Exit(1)
		}
		latestBuild = r.result
	case <-latestBuildCtx.Done():
		slog.InfoContext(ctx, "Failed to get builds due to timeout.", slog.Any("error", ctx.Err()))
		os.Exit(1)
	}

	var lastUpdateTime *time.Time
	r := <-lastUpdateTimeChan
	if r.err != nil {
		slog.ErrorContext(ctx, "Failed to get last update time.", slog.String("file", outputFile), slog.Any("error", r.err))
		os.Exit(1)
	}
	lastUpdateTime = r.result

	if lastUpdateTime != nil && lastUpdateTime.After(*latestBuild.Time) {
		slog.InfoContext(ctx, "No updates!")
		os.Exit(0)
	}

	if latestBuild.Downloads == nil {
		slog.ErrorContext(ctx, "Could not find downloads in the response.")
		os.Exit(1)
	}

	download, ok := (*latestBuild.Downloads)["server:default"]

	if !ok {
		slog.ErrorContext(ctx, "Could not find application.")
		os.Exit(1)
	} else if download.Url == nil {
		slog.ErrorContext(ctx, "Could not find download url.")
		os.Exit(1)
	}

	ctx = libs.PutDownloadName(ctx, *download.Name)
	slog.InfoContext(ctx, "Found a new build.", slog.Int("build", int(*latestBuild.Id)), slog.Time("time", *latestBuild.Time))

	if latestBuild.Commits != nil && 0 < len(*latestBuild.Commits) {
		slog.InfoContext(ctx, "Changes in this build:")
		for _, change := range *latestBuild.Commits {
			slog.InfoContext(ctx, fmt.Sprintf("  %s", *change.Message))
		}
	}

	expectedHash, err := hex.DecodeString(*download.Checksums.Sha256)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to convert the given sha256 hash to bytes", slog.Any("error", err))
		os.Exit(1)
		return
	}

	slog.InfoContext(ctx, fmt.Sprintf("Downloading %s...", *download.Name))

	downloadedHash, err := downloadFile(ctx, *download.Url, outputFile, client)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to download the file", slog.Any("error", err))
		os.Exit(1)
		return
	}

	if !bytes.Equal(expectedHash, downloadedHash) {
		slog.ErrorContext(ctx, "Downloaded file is corrupted!")
		slog.ErrorContext(ctx, "Deleting downloaded file...")
		err = os.Remove(outputFile)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to remove downloaded file.", slog.Any("error", err))
		}
		os.Exit(1)
		return
	}

	slog.InfoContext(ctx, fmt.Sprintf("Latest %s %s (build %d) has been downloaded to %s!", libs.GetProjectName(), libs.GetProjectVersion(), *latestBuild.Id, outputFile))
	os.Exit(0)
}

func downloadFile(ctx context.Context, url string, outputFile string, client api.Client) (hashBytes []byte, returnErr error) {
	dist, err := os.OpenFile(outputFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)

	if err != nil {
		return nil, errors.Join(errors.New("failed to open file"), err)
	}

	defer func(dist *os.File) {
		closeErr := dist.Close()
		if closeErr != nil {
			returnErr = errors.Join(returnErr, errors.New("failed to close file"), closeErr)
		}
	}(dist)

	hash := sha256.New()

	if err = client.DownloadFile(ctx, url, io.MultiWriter(dist, hash)); err != nil {
		return nil, errors.Join(errors.New("failed to download a file"), err)
	}

	return hash.Sum(nil), nil
}

type result[T any] struct {
	result T
	err    error
}

func getLatestBuild(ctx context.Context, client api.Client, c chan<- result[api.BuildResponse]) (context.Context, context.CancelFunc) {
	timeout, cancel := context.WithTimeout(ctx, time.Duration(15)*time.Second)

	go func(c chan<- result[api.BuildResponse]) {
		resp, err := client.GetLatestBuild(timeout)
		if err != nil {
			c <- result[api.BuildResponse]{err: err}
			return
		}

		c <- result[api.BuildResponse]{result: resp}
	}(c)

	return timeout, cancel
}

func getLastUpdateTime(filepath string, c chan<- result[*time.Time]) {
	go func(c chan<- result[*time.Time]) {
		fileInfo, err := os.Stat(filepath)

		if err != nil {
			if os.IsNotExist(err) {
				c <- result[*time.Time]{}
			} else {
				c <- result[*time.Time]{err: err}
			}
		} else {
			modTime := fileInfo.ModTime()
			c <- result[*time.Time]{result: &modTime}
		}
	}(c)
}
