package libs

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

const (
	ApiServerKey      = "api-server"
	ProjectNameKey    = "project-name"
	ProjectVersionKey = "project-version"
	BuildKey          = "build"
	DownloadNameKey   = "download-name"
)

func CreateContext(apiServer string, name string, version string) context.Context {
	ctx := context.Background()

	ctx = context.WithValue(ctx, ApiServerKey, apiServer)
	ctx = context.WithValue(ctx, ProjectNameKey, name)
	ctx = context.WithValue(ctx, ProjectVersionKey, version)
	return ctx
}

func PutBuild(ctx context.Context, build int32) context.Context {
	return context.WithValue(ctx, BuildKey, strconv.Itoa(int(build)))
}

func PutDownloadName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, DownloadNameKey, name)
}

func FormatString(ctx context.Context, str string, keys ...string) (string, error) {
	result := str

	for _, key := range keys {
		v := ctx.Value(key)
		if v != nil {
			result = strings.Replace(result, "{"+key+"}", v.(string), -1)
		} else {
			return "", fmt.Errorf("the key '%s' does not exist in the context", key)
		}
	}

	return result, nil
}
