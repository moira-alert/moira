package script

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
)

// Structure that represents the Script configuration in the YAML file.
type config struct {
	Name string `mapstructure:"name"`
	Exec string `mapstructure:"exec"`
}

// Sender implements moira sender interface via script execution.
type Sender struct {
	exec   string
	logger moira.Logger
}

type scriptNotification struct {
	Events    []moira.NotificationEvent `json:"events"`
	Trigger   moira.TriggerData         `json:"trigger"`
	Contact   moira.ContactData         `json:"contact"`
	Throttled bool                      `json:"throttled"`
	Timestamp int64                     `json:"timestamp"`
}

// Init read yaml config.
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger, location *time.Location, dateTimeFormat string) error {
	var cfg config
	err := mapstructure.Decode(senderSettings, &cfg)
	if err != nil {
		return fmt.Errorf("failed to decode senderSettings to script config: %w", err)
	}

	if cfg.Name == "" {
		return fmt.Errorf("required name for sender type script")
	}
	_, _, err = parseExec(cfg.Exec)
	if err != nil {
		return err
	}
	sender.exec = cfg.Exec
	sender.logger = logger
	return nil
}

// SendEvents implements Sender interface Send.
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, plots [][]byte, throttled bool) error {
	scriptFile, args, scriptBody, err := sender.buildCommandData(events, contact, trigger, throttled)
	if err != nil {
		return err
	}
	command := exec.Command(scriptFile, args...)
	var scriptOutput bytes.Buffer
	command.Stdin = bytes.NewReader(scriptBody)
	command.Stdout = &scriptOutput
	sender.logger.Debug().
		String("script", scriptFile).
		Msg("Executing script")

	err = command.Run()
	sender.logger.Debug().
		String("script", scriptFile).
		Msg("Finished executing script")

	if err != nil {
		return fmt.Errorf("failed exec [%s] Error [%s] Output: [%s]", sender.exec, err.Error(), scriptOutput.String())
	}
	return nil
}

func (sender *Sender) buildCommandData(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, throttled bool) (scriptFile string, args []string, scriptBody []byte, err error) {
	// TODO: Remove moira.VariableTriggerName from buildExecString in 2.6
	if strings.Contains(sender.exec, moira.VariableTriggerName) {
		sender.logger.Warning().
			String("variable_name", moira.VariableTriggerName).
			Msg("Variable is deprecated and will be removed in 2.6 release")
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

func buildExecString(template string, trigger moira.TriggerData, contact moira.ContactData) string {
	templateVariables := map[string]string{
		moira.VariableContactID:    contact.ID,
		moira.VariableContactValue: contact.Value,
		moira.VariableContactType:  contact.Type,
		moira.VariableTriggerID:    trigger.ID,
		moira.VariableTriggerName:  trigger.Name,
	}
	for k, v := range templateVariables {
		template = strings.ReplaceAll(template, k, v)
	}
	return template
}
