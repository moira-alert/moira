package postgresql

import (
	"fmt"

	"github.com/lib/pq"
	"github.com/moira-alert/moira"
)

func (connector *DbConnector) GetSubscription(id string) (moira.SubscriptionData, error) {
	subscription, err := connector.getSubscriptionByID(connector.db.Replica(), id)
	if err == nil {
		return subscription, nil
	}

	return connector.getSubscriptionByID(connector.db.Master(), id)
}

func (connector *DbConnector) GetSubscriptions(subscriptionIDs []string) ([]*moira.SubscriptionData, error) {
	query := `
	SELECT subscription_id
	FROM subscription
	WHERE subscription_id = ANY($1::text[]);
	`
	rows, err := connector.db.Replica().QueryContext(connector.ctx, query, pq.Array(subscriptionIDs))
	if err != nil {
		return nil, fmt.Errorf("error on get subscriptions: %w", err)
	}
	defer rows.Close()

	subscriptions := make([]*moira.SubscriptionData, 0)
	for rows.Next() {
		var subscriptionID string
		if err := rows.Scan(&subscriptionID); err != nil {
			return nil, fmt.Errorf("error on scan subscription id: %w", err)
		}

		subscription, err := connector.getSubscriptionByID(connector.db.Replica(), subscriptionID)
		if err != nil {
			subscription, err = connector.getSubscriptionByID(connector.db.Master(), subscriptionID)
		}
		if err != nil {
			return nil, err
		}

		sub := subscription
		subscriptions = append(subscriptions, &sub)
	}

	return subscriptions, nil
}

func (connector *DbConnector) SaveSubscription(subscription *moira.SubscriptionData) error {
	tx, err := connector.db.Master().BeginTx(connector.ctx, nil)
	if err != nil {
		return fmt.Errorf("error on creating transaction: %w", err)
	}
	defer tx.Rollback()

	queryWithUser :=`
		WITH selected_user AS (
				SELECT id
				FROM users
				WHERE login = $1
		)
		INSERT INTO subscription (
			subscription_id,
			enabled,
			any_tags,
			ignore_warnings,
			ignore_recoverings,
			throttling_enabled,
			user_id,
			team_id
		)
		SELECT $2,$3,$4,$5,$6,$7,selected_user.id,NULL
		FROM selected_user
		ON CONFLICT (subscription_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			any_tags = EXCLUDED.any_tags,
			ignore_warnings = EXCLUDED.ignore_warnings,
			ignore_recoverings = EXCLUDED.ignore_recoverings,
			throttling_enabled = EXCLUDED.throttling_enabled,
			user_id = EXCLUDED.user_id,
			team_id = EXCLUDED.team_id
		RETURNING id;
	`
	queryWithTeam :=`
		WITH selected_team AS (
				SELECT id
				FROM teams
				WHERE team_id = $1
		)
		INSERT INTO subscription (
			subscription_id,
			enabled,
			any_tags,
			ignore_warnings,
			ignore_recoverings,
			throttling_enabled,
			user_id,
			team_id
		)
		SELECT $2,$3,$4,$5,$6,$7,NULL,selected_team.id
		FROM selected_team
		ON CONFLICT (subscription_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			any_tags = EXCLUDED.any_tags,
			ignore_warnings = EXCLUDED.ignore_warnings,
			ignore_recoverings = EXCLUDED.ignore_recoverings,
			throttling_enabled = EXCLUDED.throttling_enabled,
			user_id = EXCLUDED.user_id,
			team_id = EXCLUDED.team_id
		RETURNING id;
	`
	query := queryWithUser
	ownerID := subscription.User

	if subscription.TeamID != "" {
		query = queryWithTeam
		ownerID = subscription.TeamID
	}

	var insertedSubscriptionID int64

	err = tx.QueryRowContext(connector.ctx, query,
		ownerID,
		subscription.ID,
		subscription.Enabled,
		subscription.AnyTags,
		subscription.IgnoreWarnings,
		subscription.IgnoreRecoverings,
		subscription.ThrottlingEnabled,
	).Scan(&insertedSubscriptionID)

	if err != nil {
		return fmt.Errorf("error on save subscription: %w", err)
	}

	deleteQueries := []string{
		`DELETE FROM subscription_contact WHERE subscription_id = $1;`,
		`DELETE FROM subscription_schedule_days WHERE schedule_id = $1;`,
		`DELETE FROM subscription_schedule WHERE subscription_id = $1;`,
		`DELETE FROM subscription_plotting_data WHERE subscription_id = $1;`,
		`DELETE FROM subscription_tag WHERE subscription_id = $1;`,
	}
	for _, deleteQuery := range deleteQueries {
		if _, err := tx.ExecContext(connector.ctx, deleteQuery, insertedSubscriptionID); err != nil {
			return fmt.Errorf("error on save subscription: %w", err)
		}
	}

	stmtContact, err := tx.PrepareContext(connector.ctx, `
		WITH selected_contact AS (
				SELECT id
				FROM contacts
				WHERE contact_id = $1
		)
		INSERT INTO subscription_contact (
			subscription_id,
			contact_id
		)
		SELECT $2,selected_contact.id FROM selected_contact;
	`)
	if err != nil {
		return fmt.Errorf("error on save subscription: %w", err)
	}
	defer stmtContact.Close()

	for _, contact := range subscription.Contacts {
		if _, err := stmtContact.ExecContext(
			connector.ctx,
			contact,
			insertedSubscriptionID,
		); err != nil {
			return fmt.Errorf("error on save subscription: %w", err)
		}
	}

	_, err = tx.ExecContext(connector.ctx, `
		INSERT INTO subscription_schedule (
			subscription_id,
			timezone_offset,
			start_offset,
			end_offset
		)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (subscription_id) DO UPDATE SET
			timezone_offset = EXCLUDED.timezone_offset,
			start_offset = EXCLUDED.start_offset,
			end_offset = EXCLUDED.end_offset
	`,
		insertedSubscriptionID,
		subscription.Schedule.TimezoneOffset,
		subscription.Schedule.StartOffset,
		subscription.Schedule.EndOffset,
	)

	if err != nil {
		return fmt.Errorf("error on save subscription: %w", err)
	}

	stmtDay, err := tx.PrepareContext(connector.ctx, `
		INSERT INTO subscription_schedule_days (
			schedule_id,
			name,
			enabled
		)
		VALUES ($1,$2,$3)
	`)
	if err != nil {
		return fmt.Errorf("error on save subscription: %w", err)
	}
	defer stmtDay.Close()

	for _, day := range subscription.Schedule.Days {
		if _, err := stmtDay.ExecContext(
			connector.ctx,
			insertedSubscriptionID,
			day.Name,
			day.Enabled,
		); err != nil {
			return fmt.Errorf("error on save subscription: %w", err)
		}
	}

	_, err = tx.ExecContext(connector.ctx, `
	INSERT INTO subscription_plotting_data (
		subscription_id,
		enabled,
		theme
	) VALUES (
		$1,
		$2,
		$3
	)
	`, insertedSubscriptionID, subscription.Plotting.Enabled, subscription.Plotting.Theme,
	)
	if err != nil {
		return fmt.Errorf("error on save plotting data: %w", err)
	}

	_, err = tx.ExecContext(connector.ctx , `
    INSERT INTO tag(tag)
    SELECT unnest($1::text[])
    ON CONFLICT (tag) DO NOTHING
	`, subscription.Tags)
	if err != nil {
		return fmt.Errorf("error on save tags: %w", err)
	}

	rows, err := tx.QueryContext(connector.ctx, `
		SELECT id
		FROM tag
		WHERE tag = ANY($1::text[])
	`, subscription.Tags)

	if err != nil {
		return err
	}
	defer rows.Close()

	var tagIDs []int64

	for rows.Next() {
		var id int64

		if err := rows.Scan(&id); err != nil {
			return err
		}

		tagIDs = append(tagIDs, id)
	}

	stmtSubTag, err := tx.PrepareContext(connector.ctx, `
		INSERT INTO subscription_tag (
			subscription_id,
			tag_id
		)
		VALUES ($1,$2)
	`)
	if err != nil {
		return fmt.Errorf("error on save subscription: %w", err)
	}
	defer stmtSubTag.Close()

	for _, tag := range tagIDs {
		if _, err := stmtSubTag.ExecContext(
			connector.ctx,
			insertedSubscriptionID,
			tag,
		); err != nil {
			return fmt.Errorf("error on save subscription: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error on save subscription: %w", err)
	}

	return nil
}

func (connector *DbConnector) SaveSubscriptions(newSubscriptions []*moira.SubscriptionData) error {
	for _, sub := range newSubscriptions {
		err := connector.SaveSubscription(sub)
		if err != nil {
			return err
		}
	}
	return nil
}

func (connector *DbConnector) RemoveSubscription(subscriptionID string) error {
	query := `
	DELETE FROM subscription where subscription_id = $1;
	`
	_, err := connector.db.Master().ExecContext(connector.ctx, query, subscriptionID)
	if err != nil {
		return fmt.Errorf("error on delete subscription: %w", err)
	}
	return nil
}

func (connector *DbConnector) GetUserSubscriptionIDs(login string) ([]string, error) {
	query := `
	SELECT s.subscription_id
	FROM subscription s
	JOIN users u
		ON s.user_id = u.id
	WHERE u.login = $1;
	`
	rows, err := connector.db.Replica().QueryContext(connector.ctx, query, login)
	if err != nil {
		return nil, fmt.Errorf("error on get user subscription ids: %w", err)
	}
	defer rows.Close()

	subs := make([]string, 0)
	for rows.Next() {
		var subId string
		if err := rows.Scan(&subId); err != nil {
			return nil, fmt.Errorf("error on scan row: %w", err)
		}
		subs = append(subs, subId)
	}
	return subs, nil
}

func (connector *DbConnector) GetTeamSubscriptionIDs(teamID string) ([]string, error) {
	query := `
	SELECT s.subscription_id
	FROM subscription s
	JOIN teams t
		ON s.team_id = t.id
	WHERE t.team_id = $1;
	`
	rows, err := connector.db.Replica().QueryContext(connector.ctx, query, teamID)
	if err != nil {
		return nil, fmt.Errorf("error on get team subscription ids: %w", err)
	}
	defer rows.Close()

	subscriptions := make([]string, 0)
	for rows.Next() {
		var subscriptionID string
		if err := rows.Scan(&subscriptionID); err != nil {
			return nil, fmt.Errorf("error on scan team subscription id: %w", err)
		}
		subscriptions = append(subscriptions, subscriptionID)
	}

	return subscriptions, nil
}

func (connector *DbConnector) GetTagsSubscriptions(tags []string) ([]*moira.SubscriptionData, error) {
	if len(tags) == 0 {
		return []*moira.SubscriptionData{}, nil
	}

	query := `
	SELECT s.subscription_id
	FROM subscription s
	WHERE s.any_tags
	OR EXISTS (
		SELECT 1
		FROM subscription_tag st
		WHERE st.subscription_id = s.id
	)
	AND NOT EXISTS (
		SELECT 1
		FROM subscription_tag st
		JOIN tag t
			ON t.id = st.tag_id
		WHERE st.subscription_id = s.id
			AND NOT (t.tag = ANY($1::text[]))
	);
	`
	rows, err := connector.db.Replica().QueryContext(connector.ctx, query, pq.Array(tags))
	if err != nil {
		return nil, fmt.Errorf("error on get subscriptions: %w", err)
	}
	defer rows.Close()

	subscriptions := make([]*moira.SubscriptionData, 0)
	for rows.Next() {
		var subscriptionID string
		if err := rows.Scan(&subscriptionID); err != nil {
			return nil, fmt.Errorf("error on scan subscription id: %w", err)
		}

		subscription, err := connector.getSubscriptionByID(connector.db.Replica(), subscriptionID)
		if err != nil {
			subscription, err = connector.getSubscriptionByID(connector.db.Master(), subscriptionID)
		}
		if err != nil {
			return nil, err
		}

		sub := subscription
		subscriptions = append(subscriptions, &sub)
	}

	return subscriptions, nil
}

func (connector *DbConnector) getSubscriptionByID(db RDB, subscriptionID string) (moira.SubscriptionData, error) {
	query := `
	SELECT
		s.id,
		s.subscription_id,
		s.enabled,
		s.any_tags,
		s.ignore_warnings,
		s.ignore_recoverings,
		s.throttling_enabled,
		u.login,
		t.team_id
	FROM subscription s
	LEFT JOIN users u
		ON s.user_id = u.id
	LEFT JOIN teams t
		ON s.team_id = t.id
	WHERE s.subscription_id = $1
	LIMIT 1;
	`

	var (
		subID int64
		enabled bool
		anyTags bool
		ignoreWarnings bool
		ignoreRecoverings bool
		throttlingEnabled bool
		userID *string
		teamID *string
	)
	if err := db.QueryRowContext(connector.ctx, query, subscriptionID).Scan(
		&subID,
		&subscriptionID,
		&enabled,
		&anyTags,
		&ignoreWarnings,
		&ignoreRecoverings,
		&throttlingEnabled,
		&userID,
		&teamID,
	); err != nil {
		return moira.SubscriptionData{}, fmt.Errorf("error on scan subscription: %w", err)
	}

	contacts, err := connector.getContactIDsOfSubscription(db, subID)
	if err != nil {
		return moira.SubscriptionData{}, err
	}

	tags, err := connector.getSubscriptionTags(db, subID)
	if err != nil {
		return moira.SubscriptionData{}, err
	}

	scheduleData, err := connector.getSubscriptionScheduleData(db, subID)
	if err != nil {
		return moira.SubscriptionData{}, err
	}

	plottingData, err := connector.getSubscriptionPlottingData(db, subID)
	if err != nil {
		return moira.SubscriptionData{}, err
	}

	return moira.SubscriptionData{
		ID: subscriptionID,
		Contacts: contacts,
		Tags: tags,
		Schedule: scheduleData,
		Plotting: plottingData,
		Enabled: enabled,
		AnyTags: anyTags,
		IgnoreWarnings: ignoreWarnings,
		IgnoreRecoverings: ignoreRecoverings,
		ThrottlingEnabled: throttlingEnabled,
		User: moira.UseString(userID),
		TeamID: moira.UseString(teamID),
	}, nil
}

func (connector *DbConnector) getContactIDsOfSubscription(db RDB, subID int64) ([]string, error) {
	query := `
	SELECT c.contact_id
	FROM subscription_contact sc
	JOIN contacts c
		ON c.id = sc.contact_id
	WHERE sc.subscription_id = $1;
	`
	rows, err := db.QueryContext(connector.ctx, query, subID)
	if err != nil {
		return nil, fmt.Errorf("error on get contacts: %w", err)
	}
	defer rows.Close()

	contactIDs := make([]string, 0)
	for rows.Next() {
		var contactID string
		if err := rows.Scan(&contactID); err != nil {
			return nil, fmt.Errorf("error on scan contact id: %w", err)
		}
		contactIDs = append(contactIDs, contactID)
	}

	return contactIDs, nil
}

func (connector *DbConnector) getSubscriptionTags(db RDB, subID int64) ([]string, error) {
	query := `
	SELECT t.tag
	FROM subscription_tag st
	JOIN tag t
		ON t.id = st.tag_id
	WHERE st.subscription_id = $1;
	`
	rows, err := db.QueryContext(connector.ctx, query, subID)
	if err != nil {
		return nil, fmt.Errorf("error on get tags: %w", err)
	}
	defer rows.Close()

	tags := make([]string, 0)
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("error on scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

func (connector *DbConnector) getSubscriptionScheduleData(db RDB, subID int64) (moira.ScheduleData, error) {
	query := `
	SELECT timezone_offset, start_offset, end_offset
	FROM subscription_schedule
	WHERE subscription_id = $1;
	`
	var (
		timezoneOffset int64
		startOffset int64
		endOffset int64
	)
	if err := db.QueryRowContext(connector.ctx, query, subID).Scan(&timezoneOffset, &startOffset, &endOffset); err != nil {
		return moira.ScheduleData{}, fmt.Errorf("error on get schedule data: %w", err)
	}

	queryDays := `
	SELECT enabled, name
	FROM subscription_schedule_days
	WHERE schedule_id = $1;
	`
	rows, err := db.QueryContext(connector.ctx, queryDays, subID)
	if err != nil {
		return moira.ScheduleData{}, fmt.Errorf("error on get schedule days: %w", err)
	}
	defer rows.Close()

	days := make([]moira.ScheduleDataDay, 0)
	for rows.Next() {
		var (
			enabled bool
			name string
		)
		if err := rows.Scan(&enabled, &name); err != nil {
			return moira.ScheduleData{}, fmt.Errorf("error on scan schedule days: %w", err)
		}
		days = append(days, moira.ScheduleDataDay{
			Enabled: enabled,
			Name: moira.DayName(name),
		})
	}
	return moira.ScheduleData{
		Days: days,
		TimezoneOffset: timezoneOffset,
		StartOffset: startOffset,
		EndOffset: endOffset,
	}, nil
}

func (connector *DbConnector) getSubscriptionPlottingData(db RDB, subID int64) (moira.PlottingData, error) {
	query := `
	SELECT enabled, theme
	FROM subscription_plotting_data
	WHERE subscription_id = $1
	`
	var (
		enabled bool
		theme string
	)
	if err := db.QueryRowContext(connector.ctx, query, subID).Scan(&enabled, &theme); err != nil {
		return moira.PlottingData{}, fmt.Errorf("error on scan row: %w", err)
	}

	return moira.PlottingData{
		Enabled: enabled,
		Theme: theme,
	}, nil
}
