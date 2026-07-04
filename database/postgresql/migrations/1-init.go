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

CREATE TABLE IF NOT EXISTS tag (
	id SERIAL PRIMARY KEY,
	tag VARCHAR NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS subscription (
	id SERIAL PRIMARY KEY,
	subscription_id VARCHAR NOT NULL UNIQUE,
	enabled BOOLEAN NOT NULL,
	any_tags BOOLEAN NOT NULL,
	ignore_warnings BOOLEAN NOT NULL,
	ignore_recoverings BOOLEAN NOT NULL,
	throttling_enabled BOOLEAN NOT NULL,
	user_id INTEGER REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
	team_id INTEGER REFERENCES teams(id) ON UPDATE CASCADE ON DELETE CASCADE,
	CHECK (
    (user_id IS NOT NULL AND team_id IS NULL)
    OR
    (user_id IS NULL AND team_id IS NOT NULL)
	)
);
CREATE TABLE IF NOT EXISTS subscription_contact (
	subscription_id INTEGER REFERENCES subscription(id) ON UPDATE CASCADE ON DELETE CASCADE,
	contact_id INTEGER REFERENCES contacts(id) ON UPDATE CASCADE ON DELETE CASCADE,
	PRIMARY KEY (subscription_id, contact_id)
);
CREATE TABLE IF NOT EXISTS subscription_schedule (
	subscription_id INTEGER REFERENCES subscription(id) ON UPDATE CASCADE ON DELETE CASCADE PRIMARY KEY,
	timezone_offset INTEGER NOT NULL,
	start_offset INTEGER NOT NULL,
	end_offset INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS subscription_schedule_days (
	schedule_id INTEGER REFERENCES subscription_schedule(subscription_id) ON UPDATE CASCADE ON DELETE CASCADE,
	enabled BOOLEAN NOT NULL,
	name VARCHAR NOT NULL,
	PRIMARY KEY (schedule_id, name)
);
CREATE TABLE IF NOT EXISTS subscription_plotting_data (
	subscription_id INTEGER REFERENCES subscription(id) ON UPDATE CASCADE ON DELETE CASCADE PRIMARY KEY,
	enabled BOOLEAN NOT NULL,
	theme VARCHAR NOT NULL
);
CREATE TABLE IF NOT EXISTS subscription_tag (
    subscription_id INTEGER NOT NULL
        REFERENCES subscription(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,

    tag_id INTEGER REFERENCES tag(id) ON UPDATE CASCADE ON DELETE CASCADE,

    PRIMARY KEY (subscription_id, tag_id)
);
		`,
	}
}

