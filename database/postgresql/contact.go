package postgresql

import (
	"fmt"

	"github.com/lib/pq"
	"github.com/moira-alert/moira"
)

func (connector *DbConnector) GetContact(contactId string) (moira.ContactData, error) {
	query := `
SELECT
    c.type,
    c.name,
    c.value,
    c.extra_message,

    u.login,
    t.team_id

FROM contacts c
LEFT JOIN users u
    ON c.user_id = u.id
LEFT JOIN teams t
    ON c.team_id = t.id
WHERE c.contact_id = $1 LIMIT 1;
	`
	requester := func(db RDB) (moira.ContactData, error) {
		row := db.QueryRowContext(connector.ctx, query, contactId)
		var (
			contactType string
			name string
			value string
			extraMessage string
			userLogin *string
			teamId *string
		)
		err := row.Scan(&contactType, &name, &value, &extraMessage, &userLogin, &teamId)
		return moira.ContactData{
			Type: contactType,
			Name: name,
			Value: value,
			ID: contactId,
			User: moira.UseString(userLogin),
			Team: moira.UseString(teamId),
			ExtraMessage: extraMessage,
		}, err
	}

	respFromReplica, err := requester(connector.db.Replica())
	if err == nil {
		return respFromReplica, nil
	}

	respFromMaster, err := requester(connector.db.Master())
	return respFromMaster, err
}

func (connector *DbConnector) GetContacts(contactIDs []string) ([]*moira.ContactData, error) {
	query := `
SELECT
		c.contact_id,
    c.type,
    c.name,
    c.value,
    c.extra_message,

    u.login,
    t.team_id

FROM contacts c
LEFT JOIN users u
    ON c.user_id = u.id
LEFT JOIN teams t
    ON c.team_id = t.id
WHERE c.contact_id = ANY($1);
	`

	responseFromReplica, err := connector.db.Replica().QueryContext(connector.ctx, query, pq.Array(contactIDs))
	if err != nil {
		return nil, fmt.Errorf("get contacts by ids error: %w", err)
	}

	contacts := make([]*moira.ContactData, 0)
	for responseFromReplica.Next() {
		var (
			contactId string
			contactType string
			name string
			value string
			extraMessage string
			userLogin *string
			teamId *string
		)
		err := responseFromReplica.Scan(&contactId, &contactType, &name, &value, &extraMessage, &userLogin, &teamId)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling error on get contacts: %w", err)
		}
		contacts = append(contacts, &moira.ContactData{
			Type: contactType,
			Name: name,
			Value: value,
			ID: contactId,
			User: moira.UseString(userLogin),
			Team: moira.UseString(teamId),
			ExtraMessage: extraMessage,
		})
	}
	return contacts, nil
}

func (connector *DbConnector) GetAllContacts() ([]*moira.ContactData, error) {
	query := `
SELECT
		c.contact_id,
    c.type,
    c.name,
    c.value,
    c.extra_message,

    u.login,
    t.team_id

FROM contacts c
LEFT JOIN users u
    ON c.user_id = u.id
LEFT JOIN teams t
    ON c.team_id = t.id;
	`

	responseFromReplica, err := connector.db.Replica().QueryContext(connector.ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get contacts by ids error: %w", err)
	}

	contacts := make([]*moira.ContactData, 0)
	for responseFromReplica.Next() {
		var (
			contactId string
			contactType string
			name string
			value string
			extraMessage string
			userLogin *string
			teamId *string
		)
		err := responseFromReplica.Scan(&contactId, &contactType, &name, &value, &extraMessage, &userLogin, &teamId)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling error on get all contacts: %w", err)
		}
		contacts = append(contacts, &moira.ContactData{
			Type: contactType,
			Name: name,
			Value: value,
			ID: contactId,
			User: moira.UseString(userLogin),
			Team: moira.UseString(teamId),
			ExtraMessage: extraMessage,
		})
	}
	return contacts, nil
}

func (connector *DbConnector) SaveContact(contact *moira.ContactData) error {
	if contact == nil {
		return nil
	}

	queryWithUser := `
WITH selected_user AS (
    SELECT id
    FROM users
    WHERE login = $1
)
INSERT INTO contacts (
    type,
    name,
    value,
    contact_id,
    user_id,
    team_id,
    extra_message
)
SELECT
    $2,
    $3,
    $4,
    $5,
    selected_user.id,
    NULL,
    $6
FROM selected_user;
	`
	queryWithTeam := `
INSERT INTO contacts (
    type,
    name,
    value,
    contact_id,
    user_id,
    team_id,
    extra_message
)
SELECT
    $2,
    $3,
    $4,
    $5,
    NULL,
    t.id,
    $6
FROM teams t
WHERE t.team_id = $1;
	`

	query := queryWithUser
	ownerId := contact.User
	if contact.Team != "" {
		query = queryWithTeam
		ownerId = contact.Team
	}
	_, err := connector.db.Master().ExecContext(connector.ctx, query,
		ownerId,
		contact.Type,
		contact.Name,
		contact.Value,
		contact.ID,
		contact.ExtraMessage,
	)
	if err != nil {
		return fmt.Errorf("save contact error: %w", err)
	}
	return nil
}

func (connector *DbConnector) RemoveContact(contactID string) error {
	//TODO: maybe we should use removing by setting flag instead of hard delete
	query := `
DELETE FROM contacts WHERE contact_id = $1
	`
	_, err := connector.db.Master().ExecContext(connector.ctx, query, contactID)
	if err != nil {
		return fmt.Errorf("remove contact error: %w", err)
	}
	return nil
}

func (connector *DbConnector) GetUserContactIDs(login string) ([]string, error) {
	query := `
WITH selected_user AS (
    SELECT id
    FROM users
    WHERE login = $1
		LIMIT 1
)
SELECT contact_id
FROM contacts
WHERE user_id = (
	SELECT id
	FROM selected_user
)
	`
	responseFromReplica, err := connector.db.Replica().QueryContext(connector.ctx, query, login)
	if err != nil {
		return nil, fmt.Errorf("get contacts by user login error: %w", err)
	}

	contacts := make([]string, 0)
	for responseFromReplica.Next() {
		var contactId string
		err := responseFromReplica.Scan(&contactId)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling error on get contacts ids: %w", err)
		}
		contacts = append(contacts, contactId)
	}
	return contacts, nil
}

func (connector *DbConnector) GetTeamContactIDs(teamId string) ([]string, error) {
	query := `
WITH selected_team AS (
    SELECT id
    FROM teams
    WHERE team_id = $1
		LIMIT 1
)
SELECT contact_id
FROM contacts
WHERE team_id = (
	SELECT id
	FROM selected_team
)
	`
	responseFromReplica, err := connector.db.Replica().QueryContext(connector.ctx, query, teamId)
	if err != nil {
		return nil, fmt.Errorf("get contacts by team id error: %w", err)
	}

	contacts := make([]string, 0)
	for responseFromReplica.Next() {
		var contactId string
		err := responseFromReplica.Scan(&contactId)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling error on get contacts ids: %w", err)
		}
		contacts = append(contacts, contactId)
	}
	return contacts, nil
}

