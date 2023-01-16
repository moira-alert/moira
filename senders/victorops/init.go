package victorops

import (
	"fmt"
	"time"

	"github.com/moira-alert/moira/senders/victorops/api"

	"github.com/moira-alert/moira"
)

// Sender implements moira sender interface for victorops
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

// Init loads yaml config, configures the victorops sender
func (sender *Sender) Init(senderSettings map[string]string, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var ok bool
	sender.routingURL, ok = senderSettings["routing_url"]
	if !ok {
		return fmt.Errorf("cannot read the routing url from the yaml config")
	}

	sender.imageStoreID, ok = senderSettings["image_store"]
	if !ok {
		logger.Warning().Msg("Cannot read image_store from the config, will not be able to attach plot images to events")
	} else {
		imageStore, ok := sender.ImageStores[sender.imageStoreID]
		if ok && imageStore.IsEnabled() {
			sender.imageStore = imageStore
			sender.imageStoreConfigured = true
		} else {
			logger.Warning().
				String("image_store_id", sender.imageStoreID).
				Msg("Image store specified has not been configured")
		}
	}
	sender.client = api.NewClient(sender.routingURL, nil)

	sender.frontURI = senderSettings["front_uri"]
	sender.logger = logger
	sender.location = location

	return nil
}
