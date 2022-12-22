package senders

import (
	"github.com/moira-alert/moira"
)

// ReadImageStoreConfig reads the image store config for a sender
// from its settings and confirms whether that image store
// is configured
func ReadImageStoreConfig(senderSettings map[string]string, imageStores map[string]moira.ImageStore, logger moira.Logger) (string, moira.ImageStore, bool) {
	imageStoreID, ok := senderSettings["image_store"]
	if !ok {
		logger.Warningf("Cannot read image_store from the config, will not be able to attach plot images to alerts")
		return "", nil, false
	}

	imageStore, ok := imageStores[imageStoreID]
	imageStoreConfigured := false
	if ok && imageStore.IsEnabled() {
		imageStoreConfigured = true
	} else {
		logger.Warningf("Image store specified (%s) has not been configured", imageStoreID)
		return "", nil, false
	}

	return imageStoreID, imageStore, imageStoreConfigured
}
