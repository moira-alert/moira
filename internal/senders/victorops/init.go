package victorops

import (
	"fmt"
	"time"

	moira2 "github.com/moira-alert/moira/internal/moira"

	"github.com/moira-alert/moira/internal/senders/victorops/api"
)

// Sender implements moira sender interface for victorops
type Sender struct {
	DataBase             moira2.Database
	ImageStores          map[string]moira2.ImageStore
	imageStoreID         string
	imageStore           moira2.ImageStore
	imageStoreConfigured bool
	logger               moira2.Logger
	frontURI             string
	location             *time.Location

	routingURL string
	client     *api.Client
}

// Init loads yaml config, configures the victorops sender
func (sender *Sender) Init(senderSettings map[string]string, logger moira2.Logger, location *time.Location, dateTimeFormat string) error {
	var ok bool
	sender.routingURL, ok = senderSettings["routing_url"]
	if !ok {
		return fmt.Errorf("cannot read the routing url from the yaml config")
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

	return nil
}
