package notifier

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
)

type NotificationPackage struct {
	Events     []moira_alert.EventData
	Trigger    moira_alert.TriggerData
	Contact    moira_alert.ContactData
	Throttled  bool
	FailCount  int
	DontResend bool
}

func (pkg NotificationPackage) String() string {
	return fmt.Sprintf("package of %d notifications to %s", len(pkg.Events), pkg.Contact.Value)
}
