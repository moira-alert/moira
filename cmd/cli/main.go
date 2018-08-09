package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
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
	convertDatabase                 = flag.Bool("convert-database", false, "Convert existing subscriptions")
)

// Moira version
var (
	MoiraVersion = "unknown"
	GitCommit    = "unknown"
	GoVersion    = "unknown"
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

	log, err := logging.ConfigureLog(config.LogFile, config.LogLevel, "cli")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't configure main logger: %v\n", err)
		os.Exit(1)
	}

	databaseSettings := config.Redis.GetSettings()
	dataBase := redis.NewDatabase(log, databaseSettings)

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

	if *convertDatabase {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Type to choose convertation strategy:\nu - update\nr - rollback")
		convertationStrategy, _ := reader.ReadString('\n')
		switch convertationStrategy {
		case "u", "update":
			if err := ConvertSubscriptions(dataBase, false); err != nil {
				fmt.Println(fmt.Sprintf("Can not convert existing subscriptions: %s", err.Error()))
			}
		case "r", "rollback":
			if err := ConvertSubscriptions(dataBase, true); err != nil {
				fmt.Println(fmt.Sprintf("Can not convert existing subscriptions: %s", err.Error()))
			}
		default:
			fmt.Println(fmt.Sprintf("No such option: %s", convertationStrategy))
		}
	}
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
	triggerIDs, err := dataBase.GetTriggerIDs()
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
	triggerIDs, err := dataBase.GetTriggerIDs()
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
func ConvertTaggedSubscription(database moira.Database, subscription *moira.SubscriptionData) {
	for tagInd := range subscription.Tags {
		switch subscription.Tags[tagInd] {
		case "ERROR":
			if !subscription.IgnoreWarnings {
				subscription.IgnoreWarnings = true
				subscription.Tags = append(subscription.Tags[:tagInd], subscription.Tags[tagInd+1:]...)
			}
		case "DEGRADATION", "HIGH DEGRADATION":
			if !subscription.IgnoreRecoverings {
				subscription.IgnoreRecoverings = true
				subscription.Tags = append(subscription.Tags[:tagInd], subscription.Tags[tagInd+1:]...)
			}
		}
	}
	database.SaveSubscription(subscription)
}

// ConvertUntaggedSubscription can be used in rollback if something will go wrong after Moira update.
// This method checks that subscription must ignore specific states transitions and adds required pseudo-tags to existing subscription's tags.
func ConvertUntaggedSubscription(database moira.Database, subscription *moira.SubscriptionData) {
	if subscription.IgnoreWarnings {
		subscription.Tags = append(subscription.Tags, "ERROR")
	}
	if subscription.IgnoreRecoverings {
		subscription.Tags = append(subscription.Tags, "DEGRADATION")
	}
	database.SaveSubscription(subscription)
}

// ConvertSubscriptions converts all existing tag subscriptions under specified convertation strategy
// In versions older than 2.3 Moira used to check if subscription has special pseudo-tags to ignore trigger's states transitions such as "WARN<->OK" or "ERROR->OK"
// which can be specified in subscription parameters. Starting from version 2.3 this tags are deprecated and Moira uses corresponding boolean fields instead.
func ConvertSubscriptions(database moira.Database, rollback bool) error {
	allTags, err := database.GetTagNames()
	if err != nil {
		return err
	}
	allSubscriptions, err := database.GetTagsSubscriptions(allTags)
	if err != nil {
		return err
	}
	var subscriptionsConverter func(moira.Database, *moira.SubscriptionData)
	if !rollback {
		subscriptionsConverter = ConvertTaggedSubscription
	} else {
		subscriptionsConverter = ConvertUntaggedSubscription
	}
	for _, subscription := range allSubscriptions {
		subscriptionsConverter(database, subscription)
		convertedMessage := fmt.Sprintf("Subscription %s has been succesfully converted. Tags: %s IgnoreWarnings: %t IgnoreRecoverings: %t",
			subscription.ID, strings.Join(subscription.Tags, ", "), subscription.IgnoreWarnings, subscription.IgnoreRecoverings)
		fmt.Println(convertedMessage)
	}
	return nil
}
