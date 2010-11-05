CREATE TABLE {DB_POST_TABLE} (
	num		SERIAL PRIMARY KEY,
	name		varchar(100) NOT NULL,
	trip		varchar(30) NOT NULL,
	email		varchar(320) NOT NULL,
	body		text NOT NULL,
	parent		integer REFERENCES {DB_POST_TABLE} ON DELETE CASCADE,
	created		timestamp with time zone NOT NULL,
	ip		inet NOT NULL,
	image		integer REFERENCES {DB_IMAGE_TABLE} ON DELETE SET NULL,
	image_filename	varchar(256),
	deleted		boolean NOT NULL DEFAULT FALSE,
	warned		boolean NOT NULL DEFAULT FALSE,
	banned		boolean NOT NULL DEFAULT FALSE
);
