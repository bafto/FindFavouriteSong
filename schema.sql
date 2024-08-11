CREATE TABLE IF NOT EXISTS playlist (
	id char(22) NOT NULL PRIMARY KEY,
	name varchar(128),
	url varchar(128),
	creation_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
	update_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS playlist_item (
	id char(22) NOT NULL PRIMARY KEY,
	title varchar(64),
	artists varchar(64), -- comma separated list of artists
	image varchar(64), -- URL to the image
	playlist char(22) NOT NULL REFERENCES playlist
);

CREATE TABLE IF NOT EXISTS session (
	id INTEGER PRIMARY KEY,
	playlist char(22) NOT NULL REFERENCES playlist,
	creation_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
	update_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS round (
	id INTEGER PRIMARY KEY,
	session_id INTEGER NOT NULL REFERENCES session,
	number INTEGER NOT NULL,
	creation_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
	update_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS match (
	id INTEGER PRIMARY KEY,
	round_id INTEGER NOT NULL REFERENCES round,
	winner char(22) NOT NULL REFERENCES playlist_item,
	loser char(22) NOT NULL REFERENCES playlist_item,
	creation_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
	update_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
);

PRAGMA recursive_triggers = OFF;

CREATE TRIGGER IF NOT EXISTS playlist_update_timestamp UPDATE ON playlist
	BEGIN
		UPDATE playlist SET update_timestamp = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END;

CREATE TRIGGER IF NOT EXISTS session_update_timestamp UPDATE ON session
	BEGIN
		UPDATE session SET update_timestamp = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END;

CREATE TRIGGER IF NOT EXISTS round_update_timestamp UPDATE ON round
	BEGIN
		UPDATE round SET update_timestamp = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END;

CREATE TRIGGER IF NOT EXISTS match_update_timestamp UPDATE ON match
	BEGIN
		UPDATE match SET update_timestamp = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END;