CREATE TABLE {DB_IMAGE_TABLE} (
	id		SERIAL PRIMARY KEY,
	md5		char(32) UNIQUE NOT NULL,
	filesize	integer NOT NULL,
	width		smallint NOT NULL,
	height		smallint NOT NULL,
	created		timestamp UNIQUE NOT NULL,
	deleted		boolean NOT NULL DEFAULT FALSE
);
