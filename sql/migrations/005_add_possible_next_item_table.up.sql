CREATE TABLE IF NOT EXISTS possible_next_items(
	session INTEGER NOT NULL REFERENCES session,
	playlist_item varchar(22) NOT NULL,
	lost INTEGER NOT NULL, -- BOOLEAN wether this item already is a loser
	won_round INTEGER NOT NULL, -- last round_number in which this item won
	PRIMARY KEY (session, playlist_item)
);

CREATE TRIGGER IF NOT EXISTS insert_match_trigger INSERT ON match
BEGIN
	UPDATE possible_next_items SET lost = TRUE WHERE session = new.session AND playlist_item = new.loser;
	UPDATE possible_next_items SET won_round = new.round_number WHERE session = new.session AND playlist_item = new.winner;
END;

CREATE TRIGGER IF NOT EXISTS won_trigger UPDATE OF winner ON session
BEGIN
	DELETE FROM possible_next_items WHERE session = new.id;
END;
