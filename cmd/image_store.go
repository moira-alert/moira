package cmd

import (
	"fmt"

	"github.com/moira-alert/moira"

	"github.com/mitchellh/mapstructure"

	"github.com/moira-alert/moira/imagestores/s3"
)

const (
	s3ImageStore = "s3"
)

// InitImageStores initializes the image storage provider with settings from the yaml config
func InitImageStores(imageStores map[string]map[string]string, logger moira.Logger) (map[string]moira.ImageStore, error) {
	var err error
	imageStoreMap := make(map[string]moira.ImageStore)

	for imageStoreID, imageStoreSettings := range imageStores {
		switch imageStoreID {
		case s3ImageStore:
			imageStore := &s3.ImageStore{}
			config := s3.Config{}
			err = mapstructure.Decode(imageStoreSettings, &config)
			if err = imageStore.Init(config); err != nil {
				return nil, fmt.Errorf("error while initializing image store: %s", err)
			}
			imageStoreMap[s3ImageStore] = imageStore
			logger.Infof("Image store %s initialized", imageStoreID)
		default:
			return nil, fmt.Errorf("unsupported image store type")
		}
	}
	return imageStoreMap, nil
}
