package mattermost

import "github.com/mattermost/mattermost-server/v6/model"

// Client is abstraction over model.Client4.
type Client interface {
	SetToken(token string)
	CreatePost(post *model.Post) (*model.Post, *model.Response, error)
}
