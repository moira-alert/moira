package migrations

type Migration struct {
	Number     int64
	ForwardSQL string
}

func UsersAndTeams_01() Migration {
	return Migration{
		Number:     1,
		ForwardSQL: `
CREATE TABLE IF NOT EXISTS users (
	id SERIAL PRIMARY KEY,
	login VARCHAR NOT NULL
);
CREATE TABLE IF NOT EXISTS teams (
	id SERIAL PRIMARY KEY,
	team_id VARCHAR NOT NULL,
	name VARCHAR NOT NULL,
	description VARCHAR
);
CREATE TABLE IF NOT EXISTS team_members (
	team_id INTEGER NOT NULL REFERENCES teams(id) ON UPDATE CASCADE ON DELETE CASCADE,
	user_id INTEGER NOT NULL REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
	PRIMARY KEY (team_id, user_id)
);
CREATE TABLE IF NOT EXISTS contacts (
	id SERIAL PRIMARY KEY,
	contact_id VARCHAR NOT NULL,
	type VARCHAR NOT NULL,
	name VARCHAR NOT NULL,
	value VARCHAR NOT NULL,
	user_id INTEGER REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
	team_id INTEGER REFERENCES teams(id) ON UPDATE CASCADE ON DELETE CASCADE,
	extra_message VARCHAR NOT NULL,
	CHECK (
    (user_id IS NOT NULL AND team_id IS NULL)
    OR
    (user_id IS NULL AND team_id IS NOT NULL)
	)
);
		`,
	}
}
