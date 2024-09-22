CREATE TABLE IF NOT EXISTS playlist (
	id INTEGER PRIMARY KEY,
	spotify_id varchar(22) NOT NULL,
	name varchar(128),
	url varchar(128)
);

CREATE TABLE IF NOT EXISTS playlist_item (
	id INTEGER PRIMARY KEY,
	spotify_id varchar(22) NOT NULL,
	title varchar(64),
	artists varchar(64), -- comma separated list of artists
	image varchar(64), -- URL to the image
	playlist INTEGER NOT NULL REFERENCES playlist
);

CREATE TABLE IF NOT EXISTS session (
	id INTEGER PRIMARY KEY,
	playlist INTEGER NOT NULL REFERENCES playlist
);

CREATE TABLE IF NOT EXISTS round (
	id INTEGER PRIMARY KEY,
	session_id INTEGER NOT NULL REFERENCES session,
	number INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS match (
	id INTEGER PRIMARY KEY,
	round_id INTEGER NOT NULL REFERENCES round,
	winner INTEGER NOT NULL REFERENCES playlist_item,
	loser INTEGER NOT NULL REFERENCES playlist_item
);
