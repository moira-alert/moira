package handler

const allMetricsPattern = ".*"

const (
	eventDefaultPage   = 0
	eventDefaultSize   = 100
	eventDefaultFrom   = "-inf"
	eventDefaultTo     = "+inf"
	eventDefaultMetric = allMetricsPattern
)

const (
	contactEventsDefaultFrom = "-3hour"
	contactEventsDefaultTo   = "now"
	contactEventsDefaultPage = 0
	contactEventsDefaultSize = -1
)

const (
	getAllTeamsDefaultPage          = 0
	getAllTeamsDefaultSize          = -1
	getAllTeamsDefaultRegexTemplate = ".*"
)

const (
	getTriggerNoisinessDefaultPage = 0
	getTriggerNoisinessDefaultSize = -1
	getTriggerNoisinessDefaultFrom = "-3hour"
	getTriggerNoisinessDefaultTo   = "now"
)

const (
	getContactNoisinessDefaultPage = 0
	getContactNoisinessDefaultSize = -1
	getContactNoisinessDefaultFrom = "-3hour"
	getContactNoisinessDefaultTo   = "now"
)
