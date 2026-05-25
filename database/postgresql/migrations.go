package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/moira-alert/moira/database/postgresql/migrations"
)

func (conn *DbConnector) ApplyMigrations(ctx context.Context) error {
	migrationsToApply := []migrations.Migration{
		migrations.UsersAndTeams_01(),
	}

	if err := conn.createMigrationsIfNeeded(ctx); err != nil {
		return err
	}

	lastMigrationNumber, err := conn.getLastMigrationNumber(ctx)
	if err != nil {
		return err
	}

	slices.SortFunc(migrationsToApply, func(a, b migrations.Migration) int {
		return int(a.Number - b.Number)
	})

	lastAppliedMigrationIndex := -1
	if lastMigrationNumber != 0 {
		lastAppliedMigrationIndex = slices.IndexFunc(migrationsToApply, func(m migrations.Migration) bool {
			return m.Number == lastMigrationNumber
		})
		if lastAppliedMigrationIndex == -1 {
			return fmt.Errorf("last applied migration in database with number %d not found", lastMigrationNumber)
		}
	}

	for i := range migrationsToApply[lastAppliedMigrationIndex + 1:] {
		// TODO: add logging
		migration := migrationsToApply[i]

		err := conn.applyMigration(ctx, migration)
		if err != nil {
			return fmt.Errorf("error on applying migration %d: %w", migration.Number, err)
		}
	}

	return nil
}

func (conn *DbConnector) applyMigration(ctx context.Context, migration migrations.Migration) error {
	_, err := conn.db.Master().ExecContext(ctx, migration.ForwardSQL)
	if err != nil {
		return err
	}

	query := `
INSERT INTO migrations (number, applied_at)
VALUES ($1, NOW())
	`
	_, err = conn.db.Master().ExecContext(ctx, query, migration.Number)
	return err
}

func (conn *DbConnector) getLastMigrationNumber(ctx context.Context) (int64, error) {
	query := "SELECT number FROM migrations ORDER BY applied_at LIMIT 1"

	var lastMigrationNumber int64

	err := conn.db.Master().QueryRowContext(ctx, query).Scan(&lastMigrationNumber)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return -1, fmt.Errorf("error on getting last migration number: %w", err)
	}

	return lastMigrationNumber, nil
}

func (conn *DbConnector) createMigrationsIfNeeded(ctx context.Context) error {
	query := `
CREATE TABLE IF NOT EXISTS migrations (
	id SERIAL PRIMARY KEY,
	number INTEGER NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL
)
	`
	if _, err := conn.db.Master().ExecContext(ctx, query); err != nil {
		return fmt.Errorf("error on creating migrations if needed: %w", err)
	}
	return nil
}
