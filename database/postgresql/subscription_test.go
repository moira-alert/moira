package postgresql_test

import (
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/stretchr/testify/require"
)

func TestSaveSub(t *testing.T) {
	logger, err := zerolog_adapter.GetLogger("postgresql")
	require.NoError(t, err)
	db, err := newDatabase(t.Context(), logger)
	require.NoError(t, err)

	require.NoError(t, db.ApplyMigrations(t.Context()))
	err = db.SaveSubscription(&moira.SubscriptionData{
		Contacts: []string{"contact-1"},
		Tags: []string{"a", "b", "c"},
		Schedule: moira.ScheduleData{
			Days: []moira.ScheduleDataDay{
				{
					Enabled: false,
					Name: "Mon",
				},
			},
		},
		Plotting: moira.PlottingData{
			Enabled: true,
			Theme: "dark",
		},
		ID: "sub-1",
		Enabled: true,
		AnyTags: false,
		IgnoreWarnings: false,
		IgnoreRecoverings: false,
		ThrottlingEnabled: false,
		User: "user-1",
	})
	require.NoError(t, err)
}

func TestGetSub(t *testing.T) {
	logger, err := zerolog_adapter.GetLogger("postgresql")
	require.NoError(t, err)
	db, err := newDatabase(t.Context(), logger)
	require.NoError(t, err)

	require.NoError(t, db.ApplyMigrations(t.Context()))
	sub, err := db.GetSubscription("sub-1")
	require.NoError(t, err)
	require.Equal(t, moira.SubscriptionData{
		Contacts: []string{"contact-1"},
		Tags: []string{"a", "b", "c"},
		Schedule: moira.ScheduleData{
			Days: []moira.ScheduleDataDay{
				{
					Enabled: false,
					Name: "Mon",
				},
			},
		},
		Plotting: moira.PlottingData{
			Enabled: true,
			Theme: "dark",
		},
		ID: "sub-1",
		Enabled: true,
		AnyTags: false,
		IgnoreWarnings: false,
		IgnoreRecoverings: false,
		ThrottlingEnabled: false,
		User: "user-1",
	}, sub)
}

func TestGetByTagsSub(t *testing.T) {
	logger, err := zerolog_adapter.GetLogger("postgresql")
	require.NoError(t, err)
	db, err := newDatabase(t.Context(), logger)
	require.NoError(t, err)

	require.NoError(t, db.ApplyMigrations(t.Context()))
	sub, err := db.GetTagsSubscriptions([]string{"a", "b"})
	require.NoError(t, err)
	require.Equal(t, []*moira.SubscriptionData{
		{
			Contacts: []string{"contact-1"},
			Tags: []string{"a", "b", "c"},
			Schedule: moira.ScheduleData{
				Days: []moira.ScheduleDataDay{
					{
						Enabled: false,
						Name: "Mon",
					},
				},
			},
			Plotting: moira.PlottingData{
				Enabled: true,
				Theme: "dark",
			},
			ID: "sub-1",
			Enabled: true,
			AnyTags: false,
			IgnoreWarnings: false,
			IgnoreRecoverings: false,
			ThrottlingEnabled: false,
			User: "user-1",
		},
	} , sub)
}
