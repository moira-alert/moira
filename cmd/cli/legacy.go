package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/moira-alert/moira"
)

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

func hasExpression(expr *string) bool {
	return expr != nil && *expr != ""
}
