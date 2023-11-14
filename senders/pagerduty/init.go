package pagerduty

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders"
)

// Structure that represents the PagerDuty configuration in the YAML file
type config struct {
	Name     string `mapstructure:"name"`
	Type     string `mapstructure:"type"`
	FrontURI string `mapstructure:"front_uri"`
}

// Sender implements moira sender interface for pagerduty
type Sender struct {
	clients map[string]*pagerdutyClient
}

type pagerdutyClient struct {
	ImageStores          map[string]moira.ImageStore
	imageStoreID         string
	imageStore           moira.ImageStore
	imageStoreConfigured bool
	logger               moira.Logger
	frontURI             string
	location             *time.Location
}

// Init loads yaml config, configures the pagerduty client
func (sender *Sender) Init(opts moira.InitOptions) error {
	var cfg config
	err := mapstructure.Decode(opts.SenderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to pagerduty config: %w", err)
	}

	imageStoreID, imageStore, imageStoreConfigured :=
		senders.ReadImageStoreConfig(opts.SenderSettings, opts.ImageStores, opts.Logger)

	client := &pagerdutyClient{
		frontURI:             cfg.FrontURI,
		imageStoreID:         imageStoreID,
		imageStore:           imageStore,
		imageStoreConfigured: imageStoreConfigured,
		logger:               opts.Logger,
		location:             opts.Location,
	}

	var senderIdent string
	if cfg.Name != "" {
		senderIdent = cfg.Name
	} else {
		senderIdent = cfg.Type
	}

	if sender.clients == nil {
		sender.clients = make(map[string]*pagerdutyClient)
	}

	sender.clients[senderIdent] = client

	return nil
}
