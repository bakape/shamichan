CREATE TABLE main (
	id TEXT PRIMARY KEY,
	val TEXT NOT NULL
);
INSERT INTO main (id, val) VALUES
	('version', %d),
	('config', '%s'),
	('pyu', '0');

CREATE TABLE accounts (
	id VARCHAR(20) PRIMARY KEY,
	password BYTEA NOT NULL
);

CREATE TABLE sessions (
	account VARCHAR(20) NOT NULL REFERENCES accounts ON DELETE CASCADE,
	token TEXT NOT NULL,
	expires TIMESTAMP NOT NULL,
	PRIMARY KEY (account, token)
);

CREATE TABLE images (
	APNG BOOLEAN NOT NULL,
	audio BOOLEAN NOT NULL,
	video BOOLEAN NOT NULL,
	fileType SMALLINT NOT NULL,
	thumbType SMALLINT NOT NULL,
	dims SMALLINT[4] NOT NULL,
	length INT NOT NULL,
	size INT NOT NULL,
	MD5 CHAR(22) NOT NULL,
	SHA1 CHAR(40) PRIMARY KEY
);

CREATE TABLE image_tokens (
	token CHAR(86) NOT NULL PRIMARY KEY,
	SHA1 CHAR(40) NOT NULL REFERENCES images ON DELETE CASCADE,
	expires TIMESTAMP NOT NULL
);

CREATE TABLE boards (
	readOnly BOOLEAN NOT NULL,
	textOnly BOOLEAN NOT NULL,
	forcedAnon BOOLEAN NOT NULL,
	hashCommands BOOLEAN NOT NULL,
	codeTags BOOLEAN NOT NULL,
	id VARCHAR(3) PRIMARY KEY,
	ctr BIGINT DEFAULT 0,
	created TIMESTAMP NOT NULL,
	title VARCHAR(100) NOT NULL,
	notice VARCHAR(500) NOT NULL,
	rules VARCHAR(5000) NOT NULL,
	eightball TEXT[] NOT NULL
);

CREATE TABLE staff (
	board VARCHAR(3) NOT NULL REFERENCES boards ON DELETE CASCADE,
	account VARCHAR(20) NOT NULL REFERENCES accounts ON DELETE CASCADE,
	position VARCHAR(50) NOT NULL,
	PRIMARY KEY (board, position)
);

CREATE SEQUENCE post_id;

CREATE TABLE threads (
	board VARCHAR(3) NOT NULL REFERENCES boards ON DELETE CASCADE,
	id BIGINT PRIMARY KEY,
	postCtr BIGINT NOT NULL,
	imageCtr BIGINT NOT NULL,
	bumpTime BIGINT NOT NULL,
	replyTime BIGINT NOT NULL,
	subject VARCHAR(100) NOT NULL,
	log TEXT[] NOT NULL
);
CREATE INDEX threads_board on threads (board);
CREATE INDEX bumpTime on threads (bumpTime);

CREATE TABLE posts (
	editing BOOLEAN NOT NULL,
	spoiler BOOLEAN,
	deleted BOOLEAN,
	banned BOOLEAN,
	id BIGINT PRIMARY KEY,
	op BIGINT NOT NULL REFERENCES threads ON DELETE CASCADE,
	time BIGINT NOT NULL,
	board VARCHAR(3) NOT NULL,
	trip CHAR(10),
	auth VARCHAR(20),
	SHA1 CHAR(40) REFERENCES images ON DELETE SET NULL,
	name VARCHAR(50),
	imageName VARCHAR(200),
	body VARCHAR(2000) NOT NULL,
	password BYTEA,
	ip TEXT,
	links BIGINT[][2],
	backlinks BIGINT[][2],
	commands JSON[]
);
CREATE INDEX deleted on posts (deleted);
CREATE INDEX op on posts (op);
CREATE INDEX image on posts (SHA1);
CREATE INDEX editing on posts (editing);
CREATE INDEX ip on posts (ip);
