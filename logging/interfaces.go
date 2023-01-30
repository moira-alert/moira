package logging

// EventBuilder allows to build log events with custom tags
type EventBuilder interface {
	String(key, value string) EventBuilder
	Error(err error) EventBuilder
	Int(key string, value int) EventBuilder
	Int64(key string, value int64) EventBuilder
	Interface(key string, value interface{}) EventBuilder
	Fields(fields map[string]interface{}) EventBuilder

	// Msg must be called after all tags were set
	Msg(message string)
}
