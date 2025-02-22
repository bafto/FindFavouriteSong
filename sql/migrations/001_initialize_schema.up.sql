-- PRAGMA journal_mode = WAL; -- set in data source string

CREATE TABLE IF NOT EXISTS playlist (
	id varchar(22) NOT NULL PRIMARY KEY, -- spotify id
	name varchar(128),
	url varchar(128)
);

CREATE TABLE IF NOT EXISTS playlist_item (
	id varchar(22) NOT NULL PRIMARY KEY, -- spotify id
	title varchar(64),
	artists varchar(64), -- comma separated list of artists
	image varchar(64), -- URL to the image
	playlist varchar(22) NOT NULL REFERENCES playlist -- deprectaed, removed in V004
);

CREATE TABLE IF NOT EXISTS session (
	id INTEGER PRIMARY KEY,
	playlist varchar(22) NOT NULL REFERENCES playlist,
	current_round INTEGER NOT NULL,
	user varchar(22) NOT NULL REFERENCES user,
	winner varchar(22) REFERENCES playlist_item
);

CREATE TABLE IF NOT EXISTS match (
	id INTEGER PRIMARY KEY,
	session INTEGER NOT NULL,
	round_number INTEGER NOT NULL,
	winner varchar(22) NOT NULL REFERENCES playlist_item,
	loser varchar(22) NOT NULL REFERENCES playlist_item
);

CREATE TABLE IF NOT EXISTS user (
	id varchar(22) NOT NULL PRIMARY KEY, -- spotify id
	current_session INTEGER REFERENCES session
);
