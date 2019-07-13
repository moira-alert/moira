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

// InitImageStore initializes the image storage provider with settings from the yaml config
func InitImageStore(imageStores []map[string]string, logger moira.Logger) (map[string]moira.ImageStore, error) {
	var err error
	imageStoreMap := make(map[string]moira.ImageStore)

	for _, imageStoreSettings := range imageStores {
		switch imageStoreSettings["type"] {
		case s3ImageStore:
			imageStore := &s3.ImageStore{}
			config := s3.Config{}
			err = mapstructure.Decode(imageStoreSettings, &config)
			if err = s3.Init(config, imageStore); err != nil {
				return nil, fmt.Errorf("error while initializing image store: %s", err)
			}
			imageStoreMap[s3ImageStore] = imageStore
			logger.Infof("Image store %s initialized", imageStoreSettings["type"])
		default:
			return nil, fmt.Errorf("unsupported image store type")
		}

		return imageStoreMap, nil
	}
}
