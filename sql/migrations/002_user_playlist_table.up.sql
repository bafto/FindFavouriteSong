CREATE TABLE IF NOT EXISTS playlist_added_by_user (
	user varchar(22) NOT NULL REFERENCES user,
	playlist varchar(22) NOT NULL REFERENCES playlist,
	PRIMARY KEY(user, playlist)
);
