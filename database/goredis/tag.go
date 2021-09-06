package goredis

func tagTriggersKey(tagName string) string {
	return "moira-tag-triggers:" + tagName
}

func tagSubscriptionKey(tagName string) string {
	return "moira-tag-subscriptions:" + tagName
}
