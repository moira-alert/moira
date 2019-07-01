package notifier

import (
	"fmt"

	"github.com/moira-alert/moira/imagestores/s3"
)

const (
	s3ImageStore = "s3"
)

// InitImageStore initializes the image storage provider with settings from the yaml config
func (notifier *StandardNotifier) InitImageStore(imageStoreSettings map[string]string) error {
	switch imageStoreSettings["type"] {
	case s3ImageStore:
		notifier.imageStore = &s3.ImageStore{}
	default:
		return fmt.Error("unsupported image store type")
	}
	if err := notifier.imageStore.Init(imageStoreSettings); err != nil {
		return fmt.Errorf("error while initializing image store: %s", err)
	}
	return nil
}
