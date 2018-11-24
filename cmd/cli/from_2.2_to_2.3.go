package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/moira-alert/moira"
)

func updateFrom22(logger moira.Logger, dataBase moira.Database) error {
	logger.Info("Update 2.2 -> 2.3 start")

	logger.Info("Start updating existing trigger structures into new format")
	if err := convertTriggers(dataBase, logger, false); err != nil {
		logger.Errorf("Can not update existing triggers: %s", err.Error())
	} else {
		logger.Info("Trigger structures has been successfully updated")
	}

	logger.Info("Start updating existing subscription structures into new format")
	if err := ConvertSubscriptions(dataBase, logger, false); err != nil {
		logger.Errorf("Can not update existing subscriptions: %s", err.Error())
	} else {
		logger.Info("Subscription structures has been successfully updated")
	}

	logger.Info("Update 2.2 -> 2.3 finish")
	return nil
}

func downgradeTo22(logger moira.Logger, dataBase moira.Database) error {
	logger.Info("Downgrade 2.3 -> 2.2 start")

	logger.Info("Start downgrading existing trigger structures into old format")
	if err := convertTriggers(dataBase, logger, true); err != nil {
		logger.Errorf("Can not downgrade existing triggers: %s", err.Error())
	} else {
		logger.Info("Trigger structures has been successfully downgraded")
	}

	logger.Info("Start downgrading existing subscription structures into old format")
	if err := ConvertSubscriptions(dataBase, logger, true); err != nil {
		logger.Errorf("Can not downgrade existing subscriptions: %s", err.Error())
	} else {
		logger.Info("Subscription structures has been successfully downgraded")
	}

	logger.Info("Downgrade 2.3 -> 2.2 finish")
	return nil
}

// convertTriggers converts all existing triggers  in following strategy:
// - update: Set trigger_type to one of the following options: "expression" (trigger has custom user expression) "rising" (error > warn > ok), "falling" (error < warn < ok)
// - rollback: Set trigger_type to empty string and fill omitted warn/error values
func convertTriggers(dataBase moira.Database, logger moira.Logger, rollback bool) error {
	allTriggerIDs, err := dataBase.GetLocalTriggerIDs()
	if err != nil {
		return err
	}

	allTriggers, err := dataBase.GetTriggers(allTriggerIDs)
	if err != nil {
		return err
	}

	if rollback {
		return downgradeTriggers(allTriggers, dataBase, logger)
	}

	return updateTriggers(allTriggers, dataBase, logger)
}

func updateTriggers(triggers []*moira.Trigger, dataBase moira.Database, logger moira.Logger) error {
	for _, trigger := range triggers {
		if trigger == nil {
			continue
		}
		if trigger.TriggerType == moira.RisingTrigger ||
			trigger.TriggerType == moira.FallingTrigger ||
			trigger.TriggerType == moira.ExpressionTrigger {
			logger.Debugf("Trigger %v has '%v' type - no need to convert", trigger.ID, trigger.TriggerType)
		} else if err := setProperTriggerType(trigger, logger); err != nil {
			return fmt.Errorf("trigger converter: trigger %v - could not save to Database, error: %v",
				trigger.ID, err)
		}
		logger.Debugf("Trigger %v - save to Database", trigger.ID)
		if err := dataBase.SaveTrigger(trigger.ID, trigger); err != nil {
			return err
		}
	}
	return nil
}

func downgradeTriggers(triggers []*moira.Trigger, dataBase moira.Database, logger moira.Logger) error {
	for _, trigger := range triggers {
		if trigger == nil {
			continue
		}

		if err := setProperWarnErrorExpressionValues(trigger, logger); err == nil {
			logger.Debugf("Trigger %v - save to Database", trigger.ID)
			if err2 := dataBase.SaveTrigger(trigger.ID, trigger); err2 != nil {
				return err2
			}
		} else {
			return fmt.Errorf("trigger converter: trigger %v - could not save to Database, error: %v",
				trigger.ID, err)
		}
	}
	return nil
}

// ConvertTaggedSubscription checks that subscription has deprecated pseudo-tags
// and adds corresponding fields into subscription to store actual json structure in redis
func ConvertTaggedSubscription(database moira.Database, subscription *moira.SubscriptionData) error {
	newTags := make([]string, 0)
	for _, tag := range subscription.Tags {
		switch tag {
		case stateErrorTag:
			if !subscription.IgnoreWarnings {
				subscription.IgnoreWarnings = true
			}
		case stateDegradationTag, stateHighDegradationTag:
			if !subscription.IgnoreRecoverings {
				subscription.IgnoreRecoverings = true
			}
		default:
			newTags = append(newTags, tag)
		}
	}
	subscription.Tags = newTags
	return database.SaveSubscription(subscription)
}

// ConvertUntaggedSubscription can be used in rollback if something will go wrong after Moira update.
// This method checks that subscription must ignore specific states transitions and adds required pseudo-tags to existing subscription's tags.
func ConvertUntaggedSubscription(database moira.Database, subscription *moira.SubscriptionData) error {
	if subscription.IgnoreWarnings && !subscriptionHasTag(subscription, stateErrorTag) {
		subscription.Tags = append(subscription.Tags, stateErrorTag)
	}
	if subscription.IgnoreRecoverings && !subscriptionHasTag(subscription, stateDegradationTag) {
		subscription.Tags = append(subscription.Tags, stateDegradationTag)
	}
	return database.SaveSubscription(subscription)
}

// ConvertSubscriptions converts all existing tag subscriptions under specified convertation strategy
// In versions older than 2.3 Moira used to check if subscription has special pseudo-tags to ignore trigger's states transitions such as "WARN<->OK" or "ERROR->OK"
// which can be specified in subscription parameters. Starting from version 2.3 this tags are deprecated and Moira uses corresponding boolean fields instead.
func ConvertSubscriptions(database moira.Database, logger moira.Logger, rollback bool) error {
	allTags, err := database.GetTagNames()
	if err != nil {
		return err
	}
	allSubscriptions, err := database.GetTagsSubscriptions(allTags)
	if err != nil {
		return err
	}
	var subscriptionsConverter func(moira.Database, *moira.SubscriptionData) error
	if !rollback {
		subscriptionsConverter = ConvertTaggedSubscription
	} else {
		subscriptionsConverter = ConvertUntaggedSubscription
	}
	for _, subscription := range allSubscriptions {
		if subscription != nil {
			if err := subscriptionsConverter(database, subscription); err != nil {
				convertedMessage := fmt.Sprintf("An error occurred due to convertation procees of subscription %s: %s", subscription.ID, err.Error())
				logger.Error(convertedMessage)
			} else {
				convertedMessage := fmt.Sprintf("Subscription %s has been succesfully converted. Tags: %s IgnoreWarnings: %t IgnoreRecoverings: %t",
					subscription.ID, strings.Join(subscription.Tags, ", "), subscription.IgnoreWarnings, subscription.IgnoreRecoverings)
				logger.Debug(convertedMessage)
			}
		}
	}
	return nil
}

func subscriptionHasTag(subscription *moira.SubscriptionData, tag string) bool {
	for _, subcriptionTag := range subscription.Tags {
		if subcriptionTag == tag {
			return true
		}
	}
	return false
}

func setProperTriggerType(trigger *moira.Trigger, logger moira.Logger) error {
	logger.Debugf("Trigger %v, trigger_type: '%v' - start conversion", trigger.ID, trigger.TriggerType)
	if trigger.Expression != nil && *trigger.Expression != "" {
		logger.Debugf("Trigger %v has expression '%v' - set trigger_type to '%v'...",
			trigger.ID, *trigger.Expression, moira.ExpressionTrigger)
		trigger.TriggerType = moira.ExpressionTrigger
		return nil
	}

	if trigger.WarnValue != nil && trigger.ErrorValue != nil {
		logger.Debugf("Trigger %v - warn_value: %v, error_value: %v",
			trigger.ID, trigger.WarnValue, trigger.ErrorValue)
		if *trigger.ErrorValue > *trigger.WarnValue {
			logger.Debugf("Trigger %v - set trigger_type to '%v'", trigger.ID, moira.RisingTrigger)
			trigger.TriggerType = moira.RisingTrigger
			return nil
		}
		if *trigger.ErrorValue < *trigger.WarnValue {
			logger.Debugf("Trigger %v - set trigger_type to '%v'", trigger.ID, moira.FallingTrigger)
			trigger.TriggerType = moira.FallingTrigger
			return nil
		}
		if *trigger.ErrorValue == *trigger.WarnValue {
			logger.Debugf("Trigger %v - warn_value == error_value, set trigger_type to '%v', set warn_value to 'nil'",
				trigger.ID, moira.RisingTrigger)
			trigger.TriggerType = moira.RisingTrigger
			trigger.WarnValue = nil
			return nil
		}
	}
	return fmt.Errorf("cannot update trigger %v - warn_value: %v, error_value: %v, expression: %v, trigger_type: ''",
		trigger.ID, trigger.WarnValue, trigger.ErrorValue, trigger.Expression)
}

func setProperWarnErrorExpressionValues(trigger *moira.Trigger, logger moira.Logger) error {
	expr := ""
	warnStr := "<nil>"
	errorStr := "<nil>"
	if trigger.Expression != nil {
		expr = *trigger.Expression
	}
	if trigger.WarnValue != nil {
		warnStr = strconv.FormatFloat(*trigger.WarnValue, 'f', 2, 64)
	}
	if trigger.ErrorValue != nil {
		errorStr = strconv.FormatFloat(*trigger.ErrorValue, 'f', 2, 64)
	}
	logger.Debugf("Trigger %s: warn_value: '%s', error_value: '%s', expression: '%s', trigger_type: '%s', - start conversion",
		trigger.ID, warnStr, errorStr, expr, trigger.TriggerType)
	if trigger.TriggerType == moira.ExpressionTrigger &&
		expr != "" {
		logger.Debugf("Trigger %s has expression '%s' - set trigger_type to ''", trigger.ID, expr)
		trigger.TriggerType = ""
		return nil
	}
	if trigger.WarnValue != nil && trigger.ErrorValue != nil {
		logger.Debugf("Trigger %s has warn_value '%s', error_value '%s' - set trigger_type to ''",
			trigger.ID, warnStr, errorStr)
		trigger.TriggerType = ""
		return nil
	}
	if trigger.WarnValue == nil && trigger.ErrorValue != nil {
		logger.Debugf("Trigger %s has warn_value '%s', error_value '%s' - set trigger_type to '' and update warn_value to '%s'",
			trigger.ID, warnStr, errorStr, errorStr)
		trigger.WarnValue = trigger.ErrorValue
		trigger.TriggerType = ""
		return nil
	}
	if trigger.WarnValue != nil && trigger.ErrorValue == nil {
		logger.Debugf("Trigger %s has warn_value '%s', error_value '%s' - set trigger_type to '' and update error_value to '%s'",
			trigger.ID, warnStr, errorStr, warnStr)
		trigger.ErrorValue = trigger.WarnValue
		trigger.TriggerType = ""
		return nil
	}

	return fmt.Errorf("cannot downgrade trigger %s - warn_value: '%s', error_value: '%s', expression: '%s', trigger_type: ''",
		trigger.ID, warnStr, errorStr, expr)
}
