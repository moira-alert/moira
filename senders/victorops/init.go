package victorops

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira/senders/victorops/api"

	"github.com/moira-alert/moira"
)

// Structure that represents the VictorOps configuration in the YAML file
type config struct {
	Name       string `mapstructure:"name"`
	Type       string `mapstructure:"type"`
	RoutingURL string `mapstructure:"routing_url"`
	ImageStore string `mapstructure:"image_store"`
	FrontURI   string `mapstructure:"front_uri"`
}

// Sender implements moira sender interface for victorops
type Sender struct {
	clients map[string]*victoropsClient
}

type victoropsClient struct {
	database             moira.Database
	imageStores          map[string]moira.ImageStore
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
func (sender *Sender) Init(opts moira.InitOptions) error {
	var cfg config
	err := mapstructure.Decode(opts.SenderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to victorops config: %w", err)
	}

	if cfg.RoutingURL == "" {
		return fmt.Errorf("cannot read the routing url from the yaml config")
	}

	client := &victoropsClient{
		routingURL:   cfg.RoutingURL,
		imageStoreID: cfg.ImageStore,
		client:       api.NewClient(cfg.RoutingURL, nil),
		frontURI:     cfg.FrontURI,
		logger:       opts.Logger,
		location:     opts.Location,
		database:     opts.Database,
		imageStores:  opts.ImageStores,
	}

	if client.imageStoreID == "" {
		client.logger.Warning().Msg("Cannot read image_store from the config, will not be able to attach plot images to events")
	} else {
		imageStore, ok := client.imageStores[client.imageStoreID]
		if ok && imageStore.IsEnabled() {
			client.imageStore = imageStore
			client.imageStoreConfigured = true
		} else {
			client.logger.Warning().
				String("image_store_id", client.imageStoreID).
				Msg("Image store specified has not been configured")
		}
	}

	var senderIdent string
	if cfg.Name != "" {
		senderIdent = cfg.Name
	} else {
		senderIdent = cfg.Type
	}

	if sender.clients == nil {
		sender.clients = make(map[string]*victoropsClient)
	}

	sender.clients[senderIdent] = client

	return nil
}
