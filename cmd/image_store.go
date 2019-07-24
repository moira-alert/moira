package cmd

import (
	"github.com/moira-alert/moira"

	"github.com/moira-alert/moira/image_store/s3"
)

const (
	s3ImageStore = "s3"
)

// InitImageStores initializes the image storage provider with settings from the yaml config
func InitImageStores(imageStores ImageStoreConfig, logger moira.Logger) map[string]moira.ImageStore {
	var err error
	imageStoreMap := make(map[string]moira.ImageStore)

	imageStore := &s3.ImageStore{}
	if imageStores.S3 != (s3.Config{}) {
		if err = imageStore.Init(imageStores.S3); err != nil {
			logger.Warningf("error while initializing image store: %s", err)
		} else {
			logger.Infof("Image store %s initialized", s3ImageStore)
		}
	}
	imageStoreMap[s3ImageStore] = imageStore

	return imageStoreMap
}
