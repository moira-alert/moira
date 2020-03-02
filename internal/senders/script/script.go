package script

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	moira2 "github.com/moira-alert/moira/internal/moira"
)

// Sender implements moira sender interface via script execution
type Sender struct {
	exec   string
	logger moira2.Logger
}

type scriptNotification struct {
	Events    []moira2.NotificationEvent `json:"events"`
	Trigger   moira2.TriggerData         `json:"trigger"`
	Contact   moira2.ContactData         `json:"contact"`
	Throttled bool                       `json:"throttled"`
	Timestamp int64                      `json:"timestamp"`
}

// Init read yaml config
func (sender *Sender) Init(senderSettings map[string]string, logger moira2.Logger, location *time.Location, dateTimeFormat string) error {
	if senderSettings["name"] == "" {
		return fmt.Errorf("required name for sender type script")
	}
	_, _, err := parseExec(senderSettings["exec"])
	if err != nil {
		return err
	}
	sender.exec = senderSettings["exec"]
	sender.logger = logger
	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira2.NotificationEvents, contact moira2.ContactData, trigger moira2.TriggerData, plot []byte, throttled bool) error {
	scriptFile, args, scriptBody, err := sender.buildCommandData(events, contact, trigger, throttled)
	if err != nil {
		return err
	}
	command := exec.Command(scriptFile, args...)
	var scriptOutput bytes.Buffer
	command.Stdin = bytes.NewReader(scriptBody)
	command.Stdout = &scriptOutput
	sender.logger.Debugf("Executing script: %s", scriptFile)
	err = command.Run()
	sender.logger.Debugf("Finished executing: %s", scriptFile)
	if err != nil {
		return fmt.Errorf("failed exec [%s] Error [%s] Output: [%s]", sender.exec, err.Error(), scriptOutput.String())
	}
	return nil
}

func (sender *Sender) buildCommandData(events moira2.NotificationEvents, contact moira2.ContactData, trigger moira2.TriggerData, throttled bool) (scriptFile string, args []string, scriptBody []byte, err error) {
	// TODO: Remove moira.VariableTriggerName from buildExecString in 2.6
	if strings.Contains(sender.exec, moira2.VariableTriggerName) {
		sender.logger.Warningf("%s is deprecated and will be removed in 2.6 release", moira2.VariableTriggerName)
	}
	execString := buildExecString(sender.exec, trigger, contact)
	scriptFile, args, err = parseExec(execString)
	if err != nil {
		return scriptFile, args[1:], []byte{}, err
	}
	scriptMessage := &scriptNotification{
		Events:    events,
		Trigger:   trigger,
		Contact:   contact,
		Throttled: throttled,
	}
	scriptJSON, err := json.MarshalIndent(scriptMessage, "", "\t")
	if err != nil {
		return scriptFile, args[1:], scriptJSON, fmt.Errorf("failed marshal json: %s", err.Error())
	}
	return scriptFile, args[1:], scriptJSON, nil
}

func parseExec(execString string) (scriptFile string, args []string, err error) {
	args = strings.Split(execString, " ")
	scriptFile = args[0]
	infoFile, err := os.Stat(scriptFile)
	if err != nil {
		return scriptFile, args, fmt.Errorf("file %s not found", scriptFile)
	}
	if !infoFile.Mode().IsRegular() {
		return scriptFile, args, fmt.Errorf("%s not file", scriptFile)
	}
	return scriptFile, args, nil
}

func buildExecString(template string, trigger moira2.TriggerData, contact moira2.ContactData) string {
	templateVariables := map[string]string{
		moira2.VariableContactID:    contact.ID,
		moira2.VariableContactValue: contact.Value,
		moira2.VariableContactType:  contact.Type,
		moira2.VariableTriggerID:    trigger.ID,
		moira2.VariableTriggerName:  trigger.Name,
	}
	for k, v := range templateVariables {
		template = strings.Replace(template, k, v, -1)
	}
	return template
}
