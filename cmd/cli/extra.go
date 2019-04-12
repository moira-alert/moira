package main

import (
	"strings"

	"github.com/moira-alert/moira"
)

func enablePlottingInAllSubscriptions(logger moira.Logger, database moira.Database) error {
	allTags, err := database.GetTagNames()
	if err != nil {
		return err
	}
	allSubscriptions, err := database.GetTagsSubscriptions(allTags)
	if err != nil {
		return err
	}
	for _, subscription := range allSubscriptions {
		if subscription == nil {
			continue
		}
		subscription.Plotting = moira.PlottingData{
			Enabled: true,
			Theme:   "light",
		}
		if err := database.SaveSubscription(subscription); err != nil {
			return err
		}
		logger.Debugf("Successfully enabled plotting in %s, contacts: %s", subscription.ID, strings.Join(subscription.Contacts, ", "))
	}
	return nil
}
