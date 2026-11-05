package dto

import (
	"encoding/json"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/stretchr/testify/require"
)

func TestTriggerMuteOnHolidaysMapping(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want bool
	}{
		{name: "enabled", body: `{"mute_on_holidays":true}`, want: true},
		{name: "disabled", body: `{"mute_on_holidays":false}`},
		{name: "older client omits field", body: `{}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var request TriggerModel
			require.NoError(t, json.Unmarshal([]byte(test.body), &request))
			require.Equal(t, test.want, request.ToMoiraTrigger().MuteOnHolidays)

			response := CreateTriggerModel(&moira.Trigger{MuteOnHolidays: test.want})
			require.Equal(t, test.want, response.MuteOnHolidays)
			encoded, err := json.Marshal(response)
			require.NoError(t, err)

			var fields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(encoded, &fields))
			if test.want {
				require.JSONEq(t, "true", string(fields["mute_on_holidays"]))
			} else {
				require.NotContains(t, fields, "mute_on_holidays")
			}
		})
	}
}

func TestSubscriptionMuteOnHolidaysMapping(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want bool
	}{
		{name: "enabled", body: `{"mute_on_holidays":true}`, want: true},
		{name: "disabled", body: `{"mute_on_holidays":false}`},
		{name: "older client omits field", body: `{}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var request Subscription
			require.NoError(t, json.Unmarshal([]byte(test.body), &request))
			subscription := moira.SubscriptionData(request)
			require.Equal(t, test.want, subscription.MuteOnHolidays)

			// Subscription storage uses the JSON representation of SubscriptionData.
			stored, err := json.Marshal(subscription)
			require.NoError(t, err)
			var loaded moira.SubscriptionData
			require.NoError(t, json.Unmarshal(stored, &loaded))
			require.Equal(t, test.want, loaded.MuteOnHolidays)

			encoded, err := json.Marshal(Subscription(loaded))
			require.NoError(t, err)
			var fields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(encoded, &fields))
			if test.want {
				require.JSONEq(t, "true", string(fields["mute_on_holidays"]))
			} else {
				require.NotContains(t, fields, "mute_on_holidays")
			}
		})
	}
}
