package victorops

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira/senders/victorops/api"

	"github.com/moira-alert/moira"
)

// Structure that represents the VictorOps configuration in the YAML file
type victorOps struct {
	RoutingURL string `mapstructure:"routing_url,omitempty"`
	ImageStore string `mapstructure:"image_store,omitempty"`
	FrontURI   string `mapstructure:"front_uri"`
}

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
func (sender *Sender) Init(senderSettings map[string]interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var victorops victorOps
	err := mapstructure.Decode(senderSettings, &victorops)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to victorops config: %w", err)
	}
	sender.routingURL = victorops.RoutingURL
	if sender.routingURL == "" {
		return fmt.Errorf("cannot read the routing url from the yaml config")
	}

	sender.imageStoreID = victorops.ImageStore
	if sender.imageStoreID == "" {
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

	sender.frontURI = victorops.FrontURI
	sender.logger = logger
	sender.location = location

	return nil
}
