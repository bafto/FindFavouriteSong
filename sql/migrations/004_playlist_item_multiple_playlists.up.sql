CREATE TABLE IF NOT EXISTS playlist_item_belongs_to_playlist(
	playlist_item varchar(22) NOT NULL REFERENCES playlist_item, -- spotify id
	playlist varchar(22) NOT NULL references playlist, -- spotify id
	PRIMARY KEY (playlist_item, playlist)
);

INSERT INTO playlist_item_belongs_to_playlist (playlist_item, playlist)
SELECT id, playlist FROM playlist_item;

ALTER TABLE playlist_item DROP COLUMN playlist;
