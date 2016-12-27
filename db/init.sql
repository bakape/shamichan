CREATE TABLE main (
	id TEXT
		NOT NULL
		PRIMARY KEY,
	val TEXT
		NOT NULL
);
INSERT INTO main (id, val) VALUES
	('version', $1),
	('config', $2);

CREATE TABLE accounts (
	id VARCHAR(20)
		PRIMARY KEY,
	password BYTEA
		NOT NULL
);

CREATE TABLE sessions (
	id VARCHAR(20)
		REFERENCES accounts
		ON DELETE CASCADE,
	token TEXT,
	expires BIGINT
		NOT NULL,
	PRIMARY KEY (id, token)
);
CREATE INDEX session_expiry on sessions (expires);

CREATE TABLE images (
	APNG BOOLEAN,
	audio BOOLEAN,
	video BOOLEAN,
	fileType BIT(8)
		NOT NULL,
	thumbType BIT(8)
		NOT NULL,
	dims SMALLINT[4]
		NOT NULL,
	length INT,
	size INT
		NOT NULL,
	SHA1 CHAR(40)
		PRIMARY KEY,
	MD5 CHAR(22)
		NOT NULL
);

CREATE TABLE image_tokens (
	token CHAR(32)
		PRIMARY KEY,
	SHA1 CHAR(40)
		NOT NULL
		REFERENCES images
		ON DELETE CASCADE,
	expires BIGINT
		NOT NULL
);
CREATE INDEX image_token_expiry on image_tokens (expires);

CREATE TABLE boards (
	readOnly BOOLEAN
		NOT NULL,
	textOnly BOOLEAN
		NOT NULL,
	forcedAnon BOOLEAN
		NOT NULL,
	hashCommands BOOLEAN
		NOT NULL,
	id VARCHAR(3)
		PRIMARY KEY,
	codeTags BOOLEAN
		NOT NULL,
	title VARCHAR(100)
		NOT NULL,
	notice VARCHAR(500)
		NOT NULL,
	rules VARCHAR(5000)
		NOT NULL
);

CREATE TABLE staff (
	board VARCHAR(3)
		NOT NULL
		REFERENCES boards
		ON DELETE CASCADE,
	account VARCHAR(20)
		NOT NULL
		REFERENCES accounts
		ON DELETE CASCADE,
	position VARCHAR(50)
		NOT NULL,
	PRIMARY KEY (board, position)
);

CREATE TABLE threads (
	board VARCHAR(3)
		NOT NULL
		REFERENCES boards
		ON DELETE CASCADE,
	log BYTEA[]
		NOT NULL,
	id BIGINT
		PRIMARY KEY,
	subject VARCHAR(100)
		NOT NULL
);

CREATE TABLE posts (
	editing BOOLEAN
		NOT NULL,
	deleted BOOLEAN,
	spoiler BOOLEAN,
	board VARCHAR(3)
		NOT NULL,
	ip INET
		NOT NULL,
	id BIGSERIAL
		PRIMARY KEY,
	op BIGINT
		NOT NULL
		REFERENCES threads
		ON DELETE CASCADE,
	time BIGINT
		NOT NULL,
	body VARCHAR(2000)
		NOT NULL,
	postPassword BYTEA
		NOT NULL,
	name VARCHAR(50),
	trip CHAR(10),
	auth VARCHAR(20),
	SHA1 CHAR(40)
		REFERENCES images
		ON DELETE SET NULL,
	imageName VARCHAR(200),
	commands TEXT[]
);
CREATE INDEX editing on posts (editing);
CREATE INDEX ip on posts (ip);

CREATE TABLE backlinks (
	targetBoard VARCHAR(3)
		NOT NULL,
	source BIGINT
		PRIMARY KEY
		REFERENCES posts
		ON DELETE CASCADE,
	target BIGINT
		NOT NULL
		REFERENCES posts
		ON DELETE CASCADE
);

CREATE TABLE links (
	targetBoard VARCHAR(3)
		NOT NULL,
	source BIGINT
		PRIMARY KEY
		REFERENCES posts
		ON DELETE CASCADE,
	target BIGINT
		NOT NULL
		REFERENCES posts
		ON DELETE CASCADE
);
