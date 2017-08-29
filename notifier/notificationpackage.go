package notifier

import (
	"fmt"
	"github.com/moira-alert/moira-alert"
)

//NotificationPackage represent sending data
type NotificationPackage struct {
	Events     []moira.NotificationEvent
	Trigger    moira.TriggerData
	Contact    moira.ContactData
	FailCount  int
	Throttled  bool
	DontResend bool
}

func (pkg NotificationPackage) String() string {
	return fmt.Sprintf("package of %d notifications to %s", len(pkg.Events), pkg.Contact.Value)
}
