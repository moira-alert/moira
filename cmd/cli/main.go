package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

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

	if *updateDatabaseStructures {
		fmt.Printf("Start updating existing trigger structures into new format")
		if err := ConvertTriggers(dataBase, false); err != nil {
			fmt.Printf("Can not update existing triggers: %s", err.Error())
		} else {
			fmt.Printf("Trigger structures has been sucessfully updated")
		}
	}

	if *downgradeDatabaseStructures {
		// ToDo: In future: ask which version of Moira structures use to downgrade
		fmt.Printf("Start downgrading existing trigger structures into old format")
		if err := ConvertTriggers(dataBase, true); err != nil {
			fmt.Printf("Can not downgrade existing triggers: %s", err.Error())
		} else {
			fmt.Printf("Trigger structures has been sucessfully downgraded")
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

// ConvertTriggers converts all existing triggers  in following strategy:
// - update: Set trigger_type to one of the following options: "expression" (trigger has custom user expression) "rising" (error > warn > ok), "falling" (error < warn < ok)
// - rollback: Set trigger_type to empty string and fill omitted warn/error values
func ConvertTriggers(dataBase moira.Database, rollback bool) error {
	allTriggerIDs, err := dataBase.GetTriggerIDs()
	if err != nil {
		return err
	}

	allTriggers, err := dataBase.GetTriggers(allTriggerIDs)
	if err != nil {
		return err
	}

	if rollback {
		return downgradeTriggers(allTriggers, dataBase)
	}

	return updateTriggers(allTriggers, dataBase)
}

func updateTriggers(triggers []*moira.Trigger, dataBase moira.Database) error {
	for _, trigger := range triggers {
		if trigger == nil {
			continue
		}
		if trigger.TriggerType == moira.RisingTrigger ||
			trigger.TriggerType == moira.FallingTrigger ||
			trigger.TriggerType == moira.ExpressionTrigger {
			fmt.Printf("Trigger %v has '%v' type - no need to convert", trigger.ID, trigger.TriggerType)
			continue
		}

		if err := setProperTriggerType(trigger); err == nil {
			fmt.Printf("Trigger %v - save to Database", trigger.ID)
			if err = dataBase.SaveTrigger(trigger.ID, trigger); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("trigger converter: trigger %v - could not save to Database, error: %v",
				trigger.ID, err)
		}
	}
	return nil
}

func downgradeTriggers(triggers []*moira.Trigger, dataBase moira.Database) error {
	for _, trigger := range triggers {
		if trigger == nil {
			continue
		}

		if err := setProperWarnErrorExpressionValues(trigger); err == nil {
			fmt.Printf("Trigger %v - save to Database", trigger.ID)
			if err = dataBase.SaveTrigger(trigger.ID, trigger); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("trigger converter: trigger %v - could not save to Database, error: %v",
				trigger.ID, err)
		}
	}
	return nil
}

func setProperTriggerType(trigger *moira.Trigger) error {
	fmt.Printf("Trigger %v, trigger_type: '%v' - start conversion", trigger.ID, trigger.TriggerType)
	if trigger.Expression != nil && *trigger.Expression != "" {
		fmt.Printf("Trigger %v has expression '%v' - set trigger_type to '%v'...",
			trigger.ID, *trigger.Expression, moira.ExpressionTrigger)
		trigger.TriggerType = moira.ExpressionTrigger
		return nil
	}

	if trigger.WarnValue != nil && trigger.ErrorValue != nil {
		fmt.Printf("Trigger %v - warn_value: %v, error_value: %v",
			trigger.ID, trigger.WarnValue, trigger.ErrorValue)
		if *trigger.ErrorValue > *trigger.WarnValue {
			fmt.Printf("Trigger %v - set trigger_type to '%v'", trigger.ID, moira.RisingTrigger)
			trigger.TriggerType = moira.RisingTrigger
			return nil
		}
		if *trigger.ErrorValue < *trigger.WarnValue {
			fmt.Printf("Trigger %v - set trigger_type to '%v'", trigger.ID, moira.FallingTrigger)
			trigger.TriggerType = moira.FallingTrigger
			return nil
		}
		if *trigger.ErrorValue == *trigger.WarnValue {
			fmt.Printf("Trigger %v - warn_value == error_value, set trigger_type to '%v', set warn_value to 'nil'",
				trigger.ID, moira.RisingTrigger)
			trigger.TriggerType = moira.RisingTrigger
			trigger.WarnValue = nil
			return nil
		}
	}
	return fmt.Errorf("cannot update trigger %v - warn_value: %v, error_value: %v, expression: %v, trigger_type: ''",
		trigger.ID, trigger.WarnValue, trigger.ErrorValue, trigger.Expression)
}

func setProperWarnErrorExpressionValues(trigger *moira.Trigger) error {
	fmt.Printf("Trigger %v: warn_value: %v, error_value: %v, expression: %v, trigger_type: '%v', - start conversion",
		trigger.ID, trigger.WarnValue, trigger.ErrorValue, trigger.Expression, trigger.TriggerType)
	if trigger.TriggerType == moira.ExpressionTrigger &&
		trigger.Expression != nil &&
		*trigger.Expression != "" {
		fmt.Printf("Trigger %v has expression '%v' - set trigger_type to ''", trigger.ID, trigger.Expression)
		trigger.TriggerType = ""
		return nil
	}
	if trigger.WarnValue != nil && trigger.ErrorValue != nil {
		fmt.Printf("Trigger %v has warn_value '%v', error_value '%v' - set trigger_type to ''",
			trigger.ID, trigger.WarnValue, trigger.ErrorValue)
		trigger.TriggerType = ""
		return nil
	}
	if trigger.WarnValue == nil && trigger.ErrorValue != nil {
		fmt.Printf("Trigger %v has warn_value '%v', error_value '%v' - set trigger_type to '' and update warn_value to '%v'",
			trigger.ID, trigger.WarnValue, trigger.ErrorValue, trigger.ErrorValue)
		trigger.WarnValue = trigger.ErrorValue
		trigger.TriggerType = ""
		return nil
	}
	if trigger.WarnValue != nil && trigger.ErrorValue == nil {
		fmt.Printf("Trigger %v has warn_value '%v', error_value '%v' - set trigger_type to '' and update error_value to '%v'",
			trigger.ID, trigger.WarnValue, trigger.ErrorValue, trigger.WarnValue)
		trigger.ErrorValue = trigger.WarnValue
		trigger.TriggerType = ""
		return nil
	}

	return fmt.Errorf("cannot downgrade trigger %v - warn_value: %v, error_value: %v, expression: %v, trigger_type: ''",
		trigger.ID, trigger.WarnValue, trigger.ErrorValue, trigger.Expression)
}
