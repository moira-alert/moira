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
	team_id INTEGER NOT NULL REFERENCES teams(id),
	user_id INTEGER NOT NULL REFERENCES users(id),
	PRIMARY KEY (team_id, user_id)
);
		`,
	}
}
