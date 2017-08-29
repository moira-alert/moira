//nolint
package dto

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
	"net/http"
)

type SubscriptionList struct {
	List []*moira.SubscriptionData `json:"list"`
}

func (*SubscriptionList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type Subscription moira.SubscriptionData

func (*Subscription) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (subscription *Subscription) Bind(r *http.Request) error {
	if len(subscription.Tags) == 0 {
		return fmt.Errorf("Subscription must have tags")
	}
	if len(subscription.Contacts) == 0 {
		return fmt.Errorf("Subscription must have contacts")
	}
	return nil
}
