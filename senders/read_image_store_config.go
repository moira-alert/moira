package senders

import (
	"github.com/moira-alert/moira"
)

// ReadImageStoreConfig reads the image store config for a sender
// from its settings and confirms whether that image store
// is configured.
func ReadImageStoreConfig(senderSettings any, imageStores map[string]moira.ImageStore, logger moira.Logger) (string, moira.ImageStore, bool) {
	settings, ok := senderSettings.(map[string]any)
	if !ok {
		logger.Warning().Msg("Failed conversion of senderSettings type to map[string]any")
		return "", nil, false
	}

	imageStoreID, ok := settings["image_store"]
	if !ok {
		logger.Warning().Msg("Cannot read image_store from the config, will not be able to attach plot images to alerts")
		return "", nil, false
	}

	imageStoreIDStr, ok := imageStoreID.(string)
	if !ok {
		logger.Warning().Msg("Failed to retrieve image_store from sender settings")
		return "", nil, false
	}

	imageStore, ok := imageStores[imageStoreIDStr]
	if ok && imageStore.IsEnabled() {
		return imageStoreIDStr, imageStore, true
	}

	logger.Warning().
		String("image_store_id", imageStoreIDStr).
		Msg("Image store specified has not been configured")

	return "", nil, false
}
