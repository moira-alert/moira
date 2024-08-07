package senders

import (
	"github.com/moira-alert/moira"
)

// ReadImageStoreConfig reads the image store config for a sender
// from its settings and confirms whether that image store
// is configured.
func ReadImageStoreConfig(senderSettings interface{}, imageStores map[string]moira.ImageStore, logger moira.Logger) (string, moira.ImageStore, bool) {
	settings, ok := senderSettings.(map[string]interface{})
	if !ok {
		logger.Warning().Msg("Failed conversion of senderSettings type to map[string]interface{}")
		return "", nil, false
	}

	IimageStoreID, ok := settings["image_store"]
	if !ok {
		logger.Warning().Msg("Cannot read image_store from the config, will not be able to attach plot images to alerts")
		return "", nil, false
	}

	imageStoreID, ok := IimageStoreID.(string)
	if !ok {
		logger.Warning().Msg("Failed to retrieve image_store from sender settings")
		return "", nil, false
	}

	imageStore, ok := imageStores[imageStoreID]
	imageStoreConfigured := false
	if ok && imageStore.IsEnabled() {
		imageStoreConfigured = true
	} else {
		logger.Warning().
			String("image_store_id", imageStoreID).
			Msg("Image store specified has not been configured")
		return "", nil, false
	}

	return imageStoreID, imageStore, imageStoreConfigured
}
