package mattermost

import (
	"context"

	"github.com/mattermost/mattermost/server/public/model"
)

// Client is abstraction over model.Client4.
type Client interface {
	SetToken(token string)
	CreatePost(ctx context.Context, post *model.Post) (*model.Post, *model.Response, error)
	UploadFile(ctx context.Context, data []byte, channelId string, filename string) (*model.FileUploadResponse, *model.Response, error)
}
