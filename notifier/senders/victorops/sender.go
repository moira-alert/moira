package victorops

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/notifier/senders/victorops/api"
)

// Sender implements moira sender interface for VictorOps.
// Use NewSender to create instance.
type Sender struct {
	DataBase             moira.Database
	ImageStores          map[string]moira.ImageStore
	imageStoreID         string
	imageStore           moira.ImageStore
	imageStoreConfigured bool
	logger               moira.Logger
	frontURI             string
	location             *time.Location

	routingURL string
	client     *api.Client
}

// NewSender creates Sender instance.
func NewSender(senderSettings map[string]string, logger moira.Logger, location *time.Location, imageStores map[string]moira.ImageStore) (*Sender, error) {
	sender := &Sender{
		ImageStores: imageStores,
	}

	var ok bool
	sender.routingURL, ok = senderSettings["routing_url"]
	if !ok {
		return nil, fmt.Errorf("cannot read the routing url from the yaml config")
	}

	sender.imageStoreID, ok = senderSettings["image_store"]
	if !ok {
		logger.Warningf("Cannot read image_store from the config, will not be able to attach plot images to events")
	} else {
		imageStore, ok := sender.ImageStores[sender.imageStoreID]
		if ok && imageStore.IsEnabled() {
			sender.imageStore = imageStore
			sender.imageStoreConfigured = true
		} else {
			logger.Warningf("Image store specified (%s) has not been configured", sender.imageStoreID)
		}
	}
	sender.client = api.NewClient(sender.routingURL, nil)

	sender.frontURI = senderSettings["front_uri"]
	sender.logger = logger
	sender.location = location

	return sender, nil
}
