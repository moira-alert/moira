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
	subscription.normalizeTags()
	if len(subscription.Tags) == 0 {
		return fmt.Errorf("subscription must have tags")
	}
	if len(subscription.Contacts) == 0 {
		return fmt.Errorf("subscription must have contacts")
	}
	return nil
}

func (subscription *Subscription) normalizeTags() {
	var tags = make([]string, 0)
	for _, subTag := range subscription.Tags {
		if subTag != "" {
			tags = append(tags, subTag)
		}
	}
	subscription.Tags = tags
}
