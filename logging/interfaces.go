package logging

// EventBuilder interface is an abstraction over logger.EventBuilder that allows to build log events with custom tags
type EventBuilder interface {
	String(key, value string) EventBuilder
	Error(err error) EventBuilder
	Int(key string, value int) EventBuilder
	Int64(key string, value int64) EventBuilder
	Value(key string, value interface{}) EventBuilder
	Fields(fields map[string]interface{}) EventBuilder
	Msg(message string)
}
