package reply

import (
	"encoding/json"
	"fmt"

	"github.com/moira-alert/moira/database"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
)

// Contact converts redis DB reply to moira.ContactData object
func Contact(rep redis.StringCmd) (moira.ContactData, error) {
	contact := moira.ContactData{}
	bytes, err := rep.Bytes()
	if err != nil {
		if err == redis.Nil {
			return contact, database.ErrNil
		}
		return contact, fmt.Errorf("failed to read contact: %s", err.Error())
	}
	err = json.Unmarshal(bytes, &contact)
	if err != nil {
		return contact, fmt.Errorf("failed to parse contact json %s: %s", string(bytes), err.Error())
	}
	return contact, nil
}
