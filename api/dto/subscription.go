// nolint
package dto

import (
	"fmt"
	"net/http"

	"github.com/moira-alert/moira"
)

type SubscriptionList struct {
	List []moira.SubscriptionData `json:"list"`
}

func (*SubscriptionList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type Subscription moira.SubscriptionData

func (*Subscription) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (subscription *Subscription) Bind(r *http.Request) error {
	subscription.Tags = normalizeTags(subscription.Tags)
	if len(subscription.Tags) == 0 {
		return fmt.Errorf("subscription must have tags")
	}
	if len(subscription.Contacts) == 0 {
		return fmt.Errorf("subscription must have contacts")
	}
	return nil
}

func normalizeTags(tags []string) []string {
	var normalized = make([]string, 0)
	for _, subTag := range tags {
		if subTag != "" {
			normalized = append(normalized, subTag)
		}
	}
	return normalized
}
