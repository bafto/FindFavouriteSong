CREATE TABLE IF NOT EXISTS fff.playlist (
	id char(22) NOT NULL PRIMARY KEY,
	name varchar(128),
	url varchar(128),
	creation_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
	update_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS fff.playlist_item (
	id char(22) NOT NULL PRIMARY KEY,
	title varchar(64),
	artists varchar(64), -- comma separated list of artists
	image varchar(64), -- URL to the image
	playlist char(22) NOT NULL REFERENCES playlist
);

CREATE TABLE IF NOT EXISTS fff.session (
	id INTEGER PRIMARY KEY,
	playlist char(22) NOT NULL REFERENCES playlist
	creation_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
	update_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS fff.round (
	id INTEGER PRIMARY KEY,
	session_id INTEGER NOT NULL REFERENCES session,
	number INTEGER NOT NULL
	creation_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
	update_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS fff.match (
	id INTEGER PRIMARY KEY,
	round_id INTEGER NOT NULL REFERENCES round,
	winner char(22) NOT NULL REFERENCES playlist_item,
	loser char(22) NOT NULL REFERENCES playlist_item
	creation_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
	update_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

PRAGMA recursive_triggers = OFF;

CREATE TRIGGER IF NOT EXISTS fff.playlist_update_timestamp UPDATE ON fff.playlist
	BEGIN
		UPDATE fff.playlist SET update_timestamp = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END;