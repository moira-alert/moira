package postgresql_test

import (
	"context"
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/postgresql"
	"github.com/moira-alert/moira/logging/zerolog_adapter"
	"github.com/stretchr/testify/require"
)

func newDatabase(ctx context.Context, logger moira.Logger) (*postgresql.DbConnector, error) {
	db, err := postgresql.NewDatabase(ctx, logger, postgresql.DatabaseConfig{
		Master: postgresql.ReplicaConfig{
			ConnectionString: "postgresql://user:password@localhost/postgres?connect_timeout=10",
		},
	})
	return db, err
}

func TestSaveTeam(t *testing.T) {
	logger, err := zerolog_adapter.GetLogger("postgresql")
	require.NoError(t, err)
	db, err := newDatabase(t.Context(), logger)
	require.NoError(t, err)

	require.NoError(t, db.ApplyMigrations(t.Context()))
	err = db.SaveTeam("team-1", moira.Team{
		Name: "team-1-name",
		Description: "team-1-description",
	})
	require.NoError(t, err)
}

func TestGetAllTeams(t *testing.T) {
	logger, err := zerolog_adapter.GetLogger("postgresql")
	require.NoError(t, err)
	db, err := newDatabase(t.Context(), logger)
	require.NoError(t, err)

	require.NoError(t, db.ApplyMigrations(t.Context()))
	// err = db.SaveTeam("team-1", moira.Team{
	// 	Name: "team-1-name",
	// 	Description: "team-1-description",
	// })
	teams, err := db.GetAllTeams()
	require.NoError(t, err)
	require.Equal(t, []moira.Team{
		{
			ID: "team-1",
			Name: "team-1-name",
			Description: "team-1-description",
		},
	}, teams)
}

