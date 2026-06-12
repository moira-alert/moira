package postgresql_test

import (
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/stretchr/testify/require"
)

func TestSaveContact(t *testing.T) {
	logger, err := zerolog_adapter.GetLogger("postgresql")
	require.NoError(t, err)
	db, err := newDatabase(t.Context(), logger)
	require.NoError(t, err)

	require.NoError(t, db.ApplyMigrations(t.Context()))
	err = db.SaveContact(&moira.ContactData{
		Type: "email",
		Name: "contact-1-name",
		Value: "test@mail.com",
		ID: "contact-1",
		User: "user-1",
		ExtraMessage: "",
	})
	require.NoError(t, err)
}

func TestGetContact(t *testing.T) {
	logger, err := zerolog_adapter.GetLogger("postgresql")
	require.NoError(t, err)
	db, err := newDatabase(t.Context(), logger)
	require.NoError(t, err)

	require.NoError(t, db.ApplyMigrations(t.Context()))
	contact, err := db.GetContact("contact-1")
	require.NoError(t, err)
	require.Equal(t, contact, moira.ContactData{
		Type: "email",
		Name: "contact-1-name",
		Value: "test@mail.com",
		ID: "contact-1",
		User: "user-1",
		ExtraMessage: "",
	})
}

func TestGetContacts(t *testing.T) {
	logger, err := zerolog_adapter.GetLogger("postgresql")
	require.NoError(t, err)
	db, err := newDatabase(t.Context(), logger)
	require.NoError(t, err)

	require.NoError(t, db.ApplyMigrations(t.Context()))
	contacts, err := db.GetContacts([]string{"contact-1"})
	require.NoError(t, err)
	require.Equal(t, []*moira.ContactData{
		{
			Type: "email",
			Name: "contact-1-name",
			Value: "test@mail.com",
			ID: "contact-1",
			User: "user-1",
			ExtraMessage: "",
		},
	}, contacts)
}

func TestGetAllContacts(t *testing.T) {
	logger, err := zerolog_adapter.GetLogger("postgresql")
	require.NoError(t, err)
	db, err := newDatabase(t.Context(), logger)
	require.NoError(t, err)

	require.NoError(t, db.ApplyMigrations(t.Context()))
	contacts, err := db.GetAllContacts()
	require.NoError(t, err)
	require.Equal(t, []*moira.ContactData{
		{
			Type: "email",
			Name: "contact-1-name",
			Value: "test@mail.com",
			ID: "contact-1",
			User: "user-1",
			ExtraMessage: "",
		},
	}, contacts)
}

func TestGetUserContactIDs(t *testing.T) {
	logger, err := zerolog_adapter.GetLogger("postgresql")
	require.NoError(t, err)
	db, err := newDatabase(t.Context(), logger)
	require.NoError(t, err)

	require.NoError(t, db.ApplyMigrations(t.Context()))
	contacts, err := db.GetUserContactIDs("user-1")
	require.NoError(t, err)
	require.Equal(t, []string{"contact-1"}, contacts)
}

func TestRemoveContact(t *testing.T) {
	logger, err := zerolog_adapter.GetLogger("postgresql")
	require.NoError(t, err)
	db, err := newDatabase(t.Context(), logger)
	require.NoError(t, err)

	require.NoError(t, db.ApplyMigrations(t.Context()))
	err = db.RemoveContact("contact-1")
	require.NoError(t, err)
}
