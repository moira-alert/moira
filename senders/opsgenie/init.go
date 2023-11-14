package opsgenie

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders"
	"github.com/opsgenie/opsgenie-go-sdk-v2/alert"
	"github.com/opsgenie/opsgenie-go-sdk-v2/client"
)

// Structure that represents the OpsGenie configuration in the YAML file
type config struct {
	Name     string `mapstructure:"name"`
	Type     string `mapstructure:"type"`
	APIKey   string `mapstructure:"api_key"`
	FrontURI string `mapstructure:"front_uri"`
}

// Sender implements the Sender interface for opsgenie
type Sender struct {
	clients map[string]*opsgenieClient
}

type opsgenieClient struct {
	apiKey               string
	client               *alert.Client
	logger               moira.Logger
	location             *time.Location
	ImageStores          map[string]moira.ImageStore
	imageStoreID         string
	imageStore           moira.ImageStore
	imageStoreConfigured bool
	frontURI             string
}

// Init initializes the opsgenie sender
func (sender *Sender) Init(opts moira.InitOptions) error {
	var cfg config

	err := mapstructure.Decode(opts.SenderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to opsgenie config: %w", err)
	}

	if cfg.APIKey == "" {
		return fmt.Errorf("cannot read the api_key from the sender settings")
	}

	imageStoreID, imageStore, imageStoreConfigured :=
		senders.ReadImageStoreConfig(opts.SenderSettings, opts.ImageStores, opts.Logger)

	client, err := alert.NewClient(&client.Config{
		ApiKey: cfg.APIKey,
	})
	if err != nil {
		return fmt.Errorf("error while creating opsgenie client: %s", err)
	}

	ogClient := &opsgenieClient{
		apiKey:               cfg.APIKey,
		frontURI:             cfg.FrontURI,
		logger:               opts.Logger,
		location:             opts.Location,
		client:               client,
		imageStoreID:         imageStoreID,
		imageStore:           imageStore,
		imageStoreConfigured: imageStoreConfigured,
	}

	var senderIdent string
	if cfg.Name != "" {
		senderIdent = cfg.Name
	} else {
		senderIdent = cfg.Type
	}

	if sender.clients == nil {
		sender.clients = make(map[string]*opsgenieClient)
	}

	sender.clients[senderIdent] = ogClient

	return nil
}
