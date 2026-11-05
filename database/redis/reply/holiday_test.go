package reply

import (
	"encoding/json"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/moira-alert/moira"
	"github.com/stretchr/testify/require"
)

func TestTriggerMuteOnHolidaysStorage(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want bool
	}{
		{name: "enabled", body: `{"id":"holiday-trigger","mute_on_holidays":true}`, want: true},
		{name: "disabled", body: `{"id":"holiday-trigger","mute_on_holidays":false}`},
		{name: "old trigger without flag", body: `{"id":"holiday-trigger"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			command := &redis.StringCmd{}
			command.SetVal(test.body)
			loaded, err := Trigger(command)
			require.NoError(t, err)
			require.Equal(t, test.want, loaded.MuteOnHolidays)

			encoded, err := GetTriggerBytes("holiday-trigger", &moira.Trigger{MuteOnHolidays: test.want})
			require.NoError(t, err)
			var fields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(encoded, &fields))
			if test.want {
				require.JSONEq(t, "true", string(fields["mute_on_holidays"]))
			} else {
				require.NotContains(t, fields, "mute_on_holidays")
			}

			// A previously enabled flag can be cleared and stays cleared after loading.
			loaded.MuteOnHolidays = false
			encoded, err = GetTriggerBytes(loaded.ID, &loaded)
			require.NoError(t, err)
			command.SetVal(string(encoded))
			updated, err := Trigger(command)
			require.NoError(t, err)
			require.False(t, updated.MuteOnHolidays)
		})
	}
}
