package libs

import (
	"context"
	"fmt"
	"strings"
)

type contextKey string

func (k contextKey) String() string {
	return string(k)
}

const (
	ApiServerKey      contextKey = "api-server"
	ProjectNameKey    contextKey = "project-name"
	ProjectVersionKey contextKey = "project-version"
	DownloadNameKey   contextKey = "download-name"
)

func CreateContext(apiServer string, name string, version string) context.Context {
	ctx := context.Background()

	ctx = context.WithValue(ctx, ApiServerKey, apiServer)
	ctx = context.WithValue(ctx, ProjectNameKey, name)
	ctx = context.WithValue(ctx, ProjectVersionKey, version)
	return ctx
}

func PutDownloadName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, DownloadNameKey, name)
}

func FormatString(ctx context.Context, str string, keys ...contextKey) (string, error) {
	result := str

	for _, key := range keys {
		v := ctx.Value(key)
		if v != nil {
			result = strings.ReplaceAll(result, "{"+key.String()+"}", v.(string))
		} else {
			return "", fmt.Errorf("the key '%s' does not exist in the context", key)
		}
	}

	return result, nil
}
