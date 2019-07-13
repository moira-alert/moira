package notifier

import (
	"fmt"

	"github.com/mitchellh/mapstructure"

	"github.com/moira-alert/moira/imagestores/s3"
)

const (
	s3ImageStore = "s3"
)

// InitImageStore initializes the image storage provider with settings from the yaml config
func (notifier *StandardNotifier) InitImageStore(imageStores []map[string]string) error {
	var err error
	for _, imageStoreSettings := range imageStores {
		switch imageStoreSettings["type"] {
		case s3ImageStore:
			notifier.imageStore = &s3.ImageStore{}
			config := s3.Config{}
			err = mapstructure.Decode(imageStoreSettings, &config)
			if err = s3.Init(config, notifier.imageStore); err != nil {
				return fmt.Errorf("error while initializing image store: %s", err)
			}
			notifier.logger.Infof("Image store %s initialized", imageStoreSettings["type"])
		default:
			return fmt.Errorf("unsupported image store type")
		}
		return nil
	}
}
