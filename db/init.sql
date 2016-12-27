CREATE TABLE 'main' (
  'id' TEXT NOT NULL,
  'val' TEXT NOT NULL,
  PRIMARY KEY ('id')
)
INSERT INTO 'main' ('id', 'val')
  VALUES ('version', $1);

CREATE TABLE 'accounts' (
  'id' VARCHAR(20) NOT NULL,
  'password' BYTEA(60) NOT NULL,
  'sessions' TEXT NOT NULL,
  PRIMARY KEY ('id')
)

CREATE TABLE 'images' (
  'SHA1' CHAR(40) NOT NULL,
  'APNG' BOOLEAN,
  'audio' BOOLEAN,
  'video' BOOLEAN,
  'fileType' BIT(8) NOT NULL,
  'thumbType' BIT(8) NOT NULL,
  'length' INT,
  'dims' SMALLINT[4] NOT NULL,
  'size' INT NOT NULL,
  'MD5' CHAR(22) NOT NULL,
  PRIMARY KEY ('SHA1')
)

CREATE TABLE 'image_tokens' (
  'token' CHAR(32) NOT NULL,
  'SHA1' CHAR(40) NOT NULL,
  'expires' BIGINT NOT NULL,
  PRIMARY KEY ('token'),
  INDEX 'expires' ('expires'),
  CONSTRAINT 'image'
    FOREIGN KEY ('SHA1')
    REFERENCES 'images' ('SHA1')
    ON DELETE CASCADE
)

CREATE TABLE 'boards' (
  'id' VARCHAR(3) NOT NULL,
  'readOnly' BOOLEAN NOT NULL,
  'textOnly' BOOLEAN NOT NULL,
  'forcedAnon' BOOLEAN NOT NULL,
  'hashCommands' BOOLEAN NOT NULL,
  'codeTags' BOOLEAN NOT NULL,
  'title' VARCHAR(100) NOT NULL,
  'notice' VARCHAR(500) NOT NULL,
  'rules' VARCHAR(5000) NOT NULL,
  PRIMARY KEY ('id')
)

CREATE TABLE 'staff' (
  'board' VARCHAR(3) NOT NULL,
  'position' VARCHAR(50) NOT NULL,
  'account' VARCHAR(20) NOT NULL,
  PRIMARY KEY ('board', 'position'),
  INDEX 'account' ('account'),
  CONSTRAINT 'board'
    FOREIGN KEY ('board')
    REFERENCES 'boards' ('id')
    ON DELETE CASCADE,
  CONSTRAINT 'account'
    FOREIGN KEY ('account')
    REFERENCES 'accounts' ('id')
    ON DELETE CASCADE
)

CREATE TABLE 'threads' (
  'id' BIGINT NOT NULL,
  'board' VARCHAR(3) NOT NULL,
  'subject' VARCHAR(100) NOT NULL,
  'log' BYTEA[] NOT NULL,
  PRIMARY KEY ('id'),
  INDEX 'board' ('board'),
  CONSTRAINT 'board'
    FOREIGN KEY ('board')
    REFERENCES 'boards' ('id')
    ON DELETE CASCADE
)

CREATE TABLE 'posts' (
  'id' BIGSERIAL NOT NULL,
  'op' BIGINT NOT NULL,
  'editing' BOOLEAN NOT NULL,
  'deleted' BOOLEAN,
  'spoiler' BOOLEAN,
  'board' VARCHAR(3) NOT NULL,
  'ip' INET NOT NULL,
  'time' BIGINT NOT NULL,
  'body' VARCHAR(2000) NOT NULL,
  'postPassword' BYTEA(60) NOT NULL,
  'name' VARCHAR(50),
  'trip' CHAR(10),
  'auth' VARCHAR(20),
  'SHA1' CHAR(40),
  'imageName' VARCHAR(200),
  'commands' TEXT[],
  PRIMARY KEY ('id'),
  INDEX 'op' ('op'),
  INDEX 'editing' ('editing'),
  INDEX 'board' ('board'),
  INDEX 'ip' ('ip'),
  INDEX 'SHA1' ('SHA1'),
  CONSTRAINT 'op'
    FOREIGN KEY ('op')
    REFERENCES 'threads' ('id')
    ON DELETE CASCADE,
  CONSTRAINT 'image'
    FOREIGN KEY ('SHA1')
    REFERENCES 'images' ('SHA1')
    ON DELETE SET NULL
)

CREATE TABLE 'backlinks' (
  'source' BIGINT NOT NULL,
  'target' BIGINT NOT NULL,
  'targetBoard' VARCHAR(3) NOT NULL,
  PRIMARY KEY ('source'),
  INDEX 'target' ('target'),
  CONSTRAINT 'source'
    FOREIGN KEY ('source')
    REFERENCES 'posts' ('id')
    ON DELETE CASCADE,
  CONSTRAINT 'target'
    FOREIGN KEY ('target')
    REFERENCES 'posts' ('id')
    ON DELETE CASCADE
)

CREATE TABLE 'meguca'.'links' (
  'source' BIGINT NOT NULL,
  'target' BIGINT NOT NULL,
  'targetBoard' VARCHAR(3) NOT NULL,
  PRIMARY KEY ('source'),
  INDEX 'target' ('target'),
  CONSTRAINT 'source'
    FOREIGN KEY ('source')
    REFERENCES 'posts' ('id')
    ON DELETE CASCADE,
  CONSTRAINT 'target'
    FOREIGN KEY ('target')
    REFERENCES 'posts' ('id')
    ON DELETE CASCADE
)
