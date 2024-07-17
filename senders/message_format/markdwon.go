package message_format

// MarkdownFormatter formats message using markdown syntax.
type MarkdownFormatter struct{}

func (_ MarkdownFormatter) Format(params MessageFormatterParams) string {
	return ""
}
