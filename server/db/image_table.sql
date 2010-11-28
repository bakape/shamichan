CREATE TABLE {DB_IMAGE_TABLE} (
	id		SERIAL PRIMARY KEY,
	created		timestamp with time zone UNIQUE NOT NULL,
	md5		char(32) UNIQUE NOT NULL,
	filesize	integer NOT NULL,
	ext		smallint NOT NULL,
	width		smallint NOT NULL,
	height		smallint NOT NULL,
	thumb_width	smallint NOT NULL,
	thumb_height	smallint NOT NULL,
	pinky_width	smallint NOT NULL,
	pinky_height	smallint NOT NULL
);
