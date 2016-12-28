CREATE TABLE main (
	id TEXT
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
	account VARCHAR(20)
		REFERENCES accounts
		ON DELETE CASCADE,
	token TEXT,
	expires BIGINT
		NOT NULL,
	PRIMARY KEY (account, token)
);

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
		NOT NULL,
	SHA1 CHAR(40)
		NOT NULL
		REFERENCES images
		ON DELETE CASCADE,
	expires BIGINT
		NOT NULL
);

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
		NOT NULL,
	eightball TEXT[]
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
CREATE INDEX threads_board on threads (board);

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
CREATE INDEX op on posts (op);
CREATE INDEX image on posts (SHA1);
CREATE INDEX editing on posts (editing);
CREATE INDEX ip on posts (ip);

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
CREATE INDEX links_source on links (source);
CREATE INDEX links_target on links (target);

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
CREATE INDEX backlinks_source on backlinks (source);
CREATE INDEX backlinks_target on backlinks (target);
