CREATE TABLE main (
	id TEXT PRIMARY KEY,
	val TEXT NOT NULL
);
INSERT INTO main (id, val) VALUES
	('version', %d),
	('config', '%s');

CREATE TABLE accounts (
	id VARCHAR(20) PRIMARY KEY,
	password BYTEA NOT NULL
);

CREATE TABLE sessions (
	expires BIGINT NOT NULL,
	account VARCHAR(20) NOT NULL REFERENCES accounts ON DELETE CASCADE,
	token TEXT NOT NULL,
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
	token CHAR(32) NOT NULL,
	SHA1 CHAR(40) NOT NULL REFERENCES images ON DELETE CASCADE,
	expires BIGINT NOT NULL
);

CREATE TABLE boards (
	readOnly BOOLEAN NOT NULL,
	textOnly BOOLEAN NOT NULL,
	forcedAnon BOOLEAN NOT NULL,
	hashCommands BOOLEAN NOT NULL,
	codeTags BOOLEAN NOT NULL,
	id VARCHAR(3) PRIMARY KEY,
	created BIGINT NOT NULL,
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
	replyTime BIGINT NOT NULL,
	subject VARCHAR(100) NOT NULL,
	log BYTEA[] NOT NULL
);
CREATE INDEX threads_board on threads (board);

CREATE TABLE posts (
	editing BOOLEAN NOT NULL,
	deleted BOOLEAN NOT NULL,
	spoiler BOOLEAN NOT NULL,
	banned BOOLEAN NOT NULL,
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
	postPassword BYTEA,
	commands JSON[]
);
CREATE INDEX deleted on posts (deleted);
CREATE INDEX op on posts (op);
CREATE INDEX image on posts (SHA1);
CREATE INDEX editing on posts (editing);

CREATE TABLE links (
	targetBoard VARCHAR(3) NOT NULL,
	source BIGINT PRIMARY KEY REFERENCES posts ON DELETE CASCADE,
	target BIGINT NOT NULL REFERENCES posts ON DELETE CASCADE,
	targetOP BIGINT NOT NULL
);
CREATE INDEX links_source on links (source);
CREATE INDEX links_target on links (target);

CREATE TABLE backlinks (
	targetBoard VARCHAR(3) NOT NULL,
	source BIGINT PRIMARY KEY REFERENCES posts ON DELETE CASCADE,
	target BIGINT NOT NULL REFERENCES posts ON DELETE CASCADE,
	targetOP BIGINT NOT NULL
);
CREATE INDEX backlinks_source on backlinks (source);
CREATE INDEX backlinks_target on backlinks (target);
