package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Sirohun09/dlpaper/api"
	"github.com/Sirohun09/dlpaper/libs"
	"io"
	"log/slog"
	"os"
	"time"
)

func main() {
	err := libs.ParseFlags()

	if err != nil {
		slog.Error("An error found in flags.", slog.Any("error", err))
		os.Exit(1)
		return
	}

	ctx := libs.CreateContext(libs.GetApiServer(), libs.GetProjectName(), libs.GetProjectVersion())

	client := api.NewClient(libs.GetApiServer())
	outputFile, err := libs.FormatString(ctx, libs.GetFilenameFormat(), libs.ProjectNameKey, libs.ProjectVersionKey)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to format output file.", slog.Any("error", err))
		os.Exit(1)
		return
	}

	slog.InfoContext(ctx, "Checking updates...")

	latestBuildChan := make(chan result[api.VersionBuild], 1)
	defer close(latestBuildChan)

	lastUpdateTimeChan := make(chan result[*time.Time], 1)
	defer close(lastUpdateTimeChan)

	latestBuildCtx, latestBuildCancel := getLatestBuild(ctx, client, latestBuildChan)
	defer latestBuildCancel()

	getLastUpdateTime(outputFile, lastUpdateTimeChan)

	var latestBuild api.VersionBuild

	select {
	case r := <-latestBuildChan:
		if r.err != nil {
			slog.ErrorContext(ctx, "Failed to get builds.", slog.Any("error", r.err))
			os.Exit(1)
			return
		}
		latestBuild = r.result
	case <-latestBuildCtx.Done():
		slog.InfoContext(ctx, "Failed to get builds due to timeout.", slog.Any("error", ctx.Err()))
		os.Exit(1)
		return
	}

	var lastUpdateTime *time.Time

	select {
	case r := <-lastUpdateTimeChan:
		if r.err != nil {
			slog.ErrorContext(ctx, "Failed to get last update time.", slog.String("file", outputFile), slog.Any("error", r.err))
			os.Exit(1)
			return
		}
		lastUpdateTime = r.result
	}

	ctx = libs.PutBuild(ctx, *latestBuild.Build)

	if lastUpdateTime != nil && lastUpdateTime.After(*latestBuild.Time) {
		slog.InfoContext(ctx, "No updates!")
		os.Exit(0)
		return
	}

	if latestBuild.Downloads == nil {
		slog.ErrorContext(ctx, "Could not find downloads in the response.")
		os.Exit(1)
		return
	}

	download, ok := (*latestBuild.Downloads)["application"]

	if !ok {
		slog.ErrorContext(ctx, "Could not find application.")
		os.Exit(1)
		return
	}

	ctx = libs.PutDownloadName(ctx, *download.Name)
	slog.InfoContext(ctx, "Found a new build.", slog.Int("build", int(*latestBuild.Build)), slog.Time("time", *latestBuild.Time))

	if latestBuild.Changes != nil && 0 < len(*latestBuild.Changes) {
		slog.InfoContext(ctx, "Changes in this build:")
		for _, change := range *latestBuild.Changes {
			slog.InfoContext(ctx, fmt.Sprintf("  %s", *change.Summary))
		}
	}

	expectedHash, err := hex.DecodeString(*download.Sha256)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to convert the given sha256 hash to bytes", slog.Any("error", err))
		os.Exit(1)
		return
	}

	slog.InfoContext(ctx, fmt.Sprintf("Downloading %s...", *download.Name))

	downloadedHash, err := downloadFile(ctx, outputFile, client)
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

	slog.InfoContext(ctx, fmt.Sprintf("Latest %s %s (build %d) has been downloaded to %s!", libs.GetProjectName(), libs.GetProjectVersion(), *latestBuild.Build, outputFile))
	os.Exit(0)
}

func downloadFile(ctx context.Context, outputFile string, client api.Client) (hashBytes []byte, returnErr error) {
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

	if err = client.Download(ctx, io.MultiWriter(dist, hash)); err != nil {
		return nil, errors.Join(errors.New("failed to download a file"), err)
	}

	return hash.Sum(nil), nil
}

type result[T any] struct {
	result T
	err    error
}

func getLatestBuild(ctx context.Context, client api.Client, c chan<- result[api.VersionBuild]) (context.Context, context.CancelFunc) {
	timeout, cancel := context.WithTimeout(ctx, time.Duration(15)*time.Second)

	go func(c chan<- result[api.VersionBuild]) {
		resp, err := client.GetVersionBuilds(timeout)
		if err != nil {
			c <- result[api.VersionBuild]{err: err}
			return
		}

		builds := *resp.Builds
		if len(builds) == 0 {
			c <- result[api.VersionBuild]{err: errors.New("no builds found")}
			return
		}

		c <- result[api.VersionBuild]{result: builds[len(builds)-1]}
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
