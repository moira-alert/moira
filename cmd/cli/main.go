package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/cmd"
	"github.com/moira-alert/moira/database/redis"
	"github.com/moira-alert/moira/logging/go-logging"
)

var (
	configFileName                  = flag.String("config", "/etc/moira/cli.yml", "Path to configuration file")
	printVersion                    = flag.Bool("version", false, "Print version and exit")
	printDefaultConfigFlag          = flag.Bool("default-config", false, "Print default config and exit")
	convertPythonExpressions        = flag.Bool("convert-expressions", false, "Convert python expression used in moira 1.x to govaluate expressions in moira 2.x")
	convertPythonExpression         = flag.String("convert-expression", "", "Convert python expression used in moira 1.x to govaluate expressions in moira 2.x for concrete trigger")
	getTriggerWithPythonExpressions = flag.Bool("python-expressions-triggers", false, "Get count of triggers with python expression and count of triggers, that has python expression and has not govaluate expression")
	removeBotInstanceLock           = flag.String("delete-bot-host-lock", "", "Delete bot host lock for launching bots with new distributed lock strategy. Must use for upgrade from Moira 1.x to 2.x")
	updateDatabaseStructures        = flag.Bool("update", false, "convert existing database structures into required ones for current Moira version")
	downgradeDatabaseStructures     = flag.Bool("downgrade", false, "reconvert existing database structures into required ones for previous Moira version")
	markTriggersAsUnused            = flag.Bool("mark-triggers-unused", false, "fill database with trigger IDs not used in any subscription")
)

// Moira version
var (
	MoiraVersion = "unknown"
	GitCommit    = "unknown"
	GoVersion    = "unknown"
)

const (
	stateErrorTag           = "ERROR"
	stateDegradationTag     = "DEGRADATION"
	stateHighDegradationTag = "HIGH DEGRADATION"
)

func main() {
	flag.Parse()
	if *printVersion {
		fmt.Println("Moira - alerting system based on graphite data")
		fmt.Println("Version:", MoiraVersion)
		fmt.Println("Git Commit:", GitCommit)
		fmt.Println("Go Version:", GoVersion)
		os.Exit(0)
	}

	config := getDefault()
	if *printDefaultConfigFlag {
		cmd.PrintConfig(config)
		os.Exit(0)
	}

	err := cmd.ReadConfig(*configFileName, &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't read settings: %v\n", err)
		os.Exit(1)
	}

	logger, err := logging.ConfigureLog(config.LogFile, config.LogLevel, "cli")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't configure main logger: %v\n", err)
		os.Exit(1)
	}

	databaseSettings := config.Redis.GetSettings()
	dataBase := redis.NewDatabase(logger, databaseSettings)

	if *convertPythonExpressions {
		if err := ConvertPythonExpressions(dataBase); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to convert: %v", err)
			os.Exit(1)
		}
	}

	if *getTriggerWithPythonExpressions {
		if err := GetTriggerWithPythonExpressions(dataBase); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to convert: %v", err)
			os.Exit(1)
		}
	}

	if *removeBotInstanceLock != "" {
		if err := RemoveBotInstanceLock(dataBase, *removeBotInstanceLock); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to convert: %v", err)
			os.Exit(1)
		}
	}

	if *convertPythonExpression != "" {
		if err := ConvertPythonExpression(dataBase, *convertPythonExpression); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to convert: %v", err)
			os.Exit(1)
		}
	}

	if *updateDatabaseStructures {
		logger.Info("Start updating existing trigger structures into new format")
		if err := ConvertTriggers(dataBase, logger, false); err != nil {
			logger.Errorf("Can not update existing triggers: %s", err.Error())
		} else {
			logger.Info("Trigger structures has been sucessfully updated")
		}
		logger.Info("Start updating existing subscription structures into new format")
		if err := ConvertSubscriptions(dataBase, logger, false); err != nil {
			logger.Errorf("Can not update existing subscriptions: %s", err.Error())
		} else {
			logger.Info("Subscription structures has been sucessfully updated")
		}
	}

	if *downgradeDatabaseStructures {
		// ToDo: In future: ask which version of Moira structures use to downgrade
		logger.Info("Start downgrading existing trigger structures into old format")
		if err := ConvertTriggers(dataBase, logger, true); err != nil {
			logger.Errorf("Can not downgrade existing triggers: %s", err.Error())
		} else {
			logger.Info("Trigger structures has been sucessfully downgraded")
		}
		logger.Info("Start downgrading existing subscription structures into old format")
		if err := ConvertSubscriptions(dataBase, logger, true); err != nil {
			logger.Errorf("Can not downgrade existing subscriptions: %s", err.Error())
		} else {
			logger.Info("Subscription structures has been sucessfully downgraded")
		}
	}

	if *markTriggersAsUnused {
		logger.Info("Start marking unused triggers")
		if err := resaveTriggers(dataBase); err == nil {
			logger.Info("Unused triggers are marked")
		} else {
			logger.Errorf("Can not mark triggers unused: %s", err.Error())
		}
	}
}

func resaveTriggers(database moira.Database) error {
	allTriggerIDs, err := database.GetAllTriggerIDs()
	if err != nil {
		return err
	}

	allTriggers, err := database.GetTriggers(allTriggerIDs)
	if err != nil {
		return err
	}

	for _, trigger := range allTriggers {
		if trigger != nil {
			err = database.SaveTrigger(trigger.ID, trigger)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// RemoveBotInstanceLock - in Moira 2.0 we switch from host-based single instance telegram-bot run lock
// to distributed lock, it allowed us to run moira in docker containers without fear that the bot will tied to the host name
func RemoveBotInstanceLock(dataBase moira.Database, botName string) error {
	fmt.Println(fmt.Sprintf("Deleting bot-host-lock for bot '%s' started", botName))
	err := dataBase.RemoveUser(botName, "moira-bot-host")
	if err != nil {
		return err
	}
	fmt.Println(fmt.Sprintf("Bot host lock for bot '%s' sucessfully deleted", botName))
	return nil
}

// GetTriggerWithPythonExpressions iterate by all triggers in system and print triggers
// count with python expressions and triggers count with govaluate expressions, used in Moira 2.0
func GetTriggerWithPythonExpressions(dataBase moira.Database) error {
	fmt.Println("Getting triggers expressions statistic started")
	triggerIDs, err := dataBase.GetAllTriggerIDs()
	if err != nil {
		return err
	}

	triggers, err := dataBase.GetTriggers(triggerIDs)
	if err != nil {
		return err
	}

	triggerWithPythonExpressions := 0
	triggerWithExpressions := 0
	doesNotConvertedTriggers := 0

	for _, trigger := range triggers {
		if trigger == nil {
			continue
		}
		if hasExpression(trigger.PythonExpression) {
			triggerWithPythonExpressions++
		}
		if hasExpression(trigger.Expression) {
			triggerWithExpressions++
		}
		if hasExpression(trigger.PythonExpression) && !hasExpression(trigger.Expression) {
			doesNotConvertedTriggers++
		}
	}

	fmt.Println(fmt.Sprintf("Triggers with python expressions: %v", triggerWithPythonExpressions))
	fmt.Println(fmt.Sprintf("Triggers with govaluate expressions: %v", triggerWithExpressions))
	fmt.Println(fmt.Sprintf("Triggers without converted expressions: %v", doesNotConvertedTriggers))
	return nil
}

func hasExpression(expr *string) bool {
	return expr != nil && *expr != ""
}

// ConvertPythonExpression used in moira 1.x to govaluate expressions in moira 2.x
// Old python expression contains in redis trigger field 'expression'
// Now new expression contains in redis field 'expr'.
// Only for one trigger
func ConvertPythonExpression(dataBase moira.Database, triggerID string) error {
	trigger, err := dataBase.GetTrigger(triggerID)
	if err != nil {
		return err
	}

	pythonExpression := trigger.PythonExpression
	if !hasExpression(pythonExpression) {
		return fmt.Errorf("Trigger has not python expression")
	}
	fmt.Println(fmt.Sprintf("Python Expression: %s", *pythonExpression))
	expression := trigger.Expression
	if hasExpression(expression) {
		fmt.Println(fmt.Sprintf("Expression: %s", *expression))
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter new expression: ")
	expr, _ := reader.ReadString('\n')
	trigger.Expression = &expr
	dataBase.SaveTrigger(trigger.ID, &trigger)
	return nil
}

// ConvertPythonExpressions used in moira 1.x to govaluate expressions in moira 2.x
// Old python expression contains in redis trigger field 'expression'
// Now new expression contains in redis field 'expr'.
func ConvertPythonExpressions(dataBase moira.Database) error {
	fmt.Println("Convert expressions started")
	triggerIDs, err := dataBase.GetAllTriggerIDs()
	if err != nil {
		return err
	}

	triggers, err := dataBase.GetTriggers(triggerIDs)
	if err != nil {
		return err
	}

	for _, trigger := range triggers {
		if trigger != nil {
			pythonExpression := trigger.PythonExpression
			if !hasExpression(pythonExpression) {
				continue
			}
			expression := trigger.Expression
			if hasExpression(expression) {
				fmt.Println(fmt.Sprintf("Found trigger with python expression and expression, triggerID: %s", trigger.ID))
				fmt.Println(fmt.Sprintf("Python expression: %s", *pythonExpression))
				fmt.Println(fmt.Sprintf("Expression: %s", *expression))
				fmt.Println()
			} else {
				fmt.Println(fmt.Sprintf("Found trigger with python expression and empty expression, triggerID: %s", trigger.ID))
				fmt.Println(fmt.Sprintf("Python expression: %s", *pythonExpression))
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("Enter new expression: ")
				expr, _ := reader.ReadString('\n')
				trigger.Expression = &expr
				dataBase.SaveTrigger(trigger.ID, trigger)
				fmt.Println()
			}
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

// ConvertTriggers converts all existing triggers  in following strategy:
// - update: Set trigger_type to one of the following options: "expression" (trigger has custom user expression) "rising" (error > warn > ok), "falling" (error < warn < ok)
// - rollback: Set trigger_type to empty string and fill omitted warn/error values
func ConvertTriggers(dataBase moira.Database, logger moira.Logger, rollback bool) error {
	allTriggerIDs, err := dataBase.GetTriggerIDs()
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

func subscriptionHasTag(subscription *moira.SubscriptionData, tag string) bool {
	for _, subcriptionTag := range subscription.Tags {
		if subcriptionTag == tag {
			return true
		}
	}
	return false
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
