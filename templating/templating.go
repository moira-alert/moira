package templating

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"
)

const eventTimeFormat = "2006-01-02 15:04:05"

type notification struct {
	Trigger trigger
	Events  []Event
}

type Event struct {
	Metric         string
	MetricElements []string
	Timestamp      int64
	Value          *float64
	State          string
}

func date(unixTime int64) string {
	return time.Unix(unixTime, 0).Format(eventTimeFormat)
}

func formatDate(unixTime int64, format string) string {
	return time.Unix(unixTime, 0).Format(format)
}

func (event Event) TimestampDecrease(second int64) int64 {
	return event.Timestamp - second
}

func (event Event) TimestampIncrease(second int64) int64 {
	return event.Timestamp + second
}

type trigger struct {
	Name string `json:"name"`
}

func Populate(name, description string, events []Event) (desc string, err error) {
	defer func() {
		if errRecover := recover(); errRecover != nil {
			desc = description
			err = fmt.Errorf("PANIC in populate: %v, Trigger name: %s, desc: %s, events:%#v",
				err, name, description, events)
		}
	}()

	buffer := bytes.Buffer{}
	funcMap := template.FuncMap{
		"date":              date,
		"formatDate":        formatDate,
		"stringsReplace":    strings.Replace,
		"stringsToLower":    strings.ToLower,
		"stringsToUpper":    strings.ToUpper,
		"stringsTrimPrefix": strings.TrimPrefix,
		"stringsTrimSuffix": strings.TrimSuffix,
		"stringsSplit":      strings.Split,
		"stringsTrim": 	     strings.Trim
	}

	dataToExecute := notification{
		Trigger: trigger{Name: name},
		Events:  events,
	}

	triggerTemplate := template.New("populate-description").Funcs(funcMap)
	triggerTemplate, err = triggerTemplate.Parse(description)
	if err != nil {
		return description, err
	}

	err = triggerTemplate.Execute(&buffer, dataToExecute)
	if err != nil {
		return description, err
	}

	return strings.TrimSpace(buffer.String()), nil
}
