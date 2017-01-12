package db

import "database/sql"

// Stores generated prepared statements
var prepared = make(map[string]*sql.Stmt, len(protoPrepared))

// Queries to be converted into prepared statements
var protoPrepared = map[string]string{
	"writePost": `
		INSERT INTO posts (
			editing, spoiler, id, board, op, time, body, name, trip, auth,
			password, ip, SHA1, imageName, links, backlinks, commands
		) VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,

	"writeImage": `
		INSERT INTO images (
			APNG, audio, video, fileType, thumbType, dims, length, size, MD5,
			SHA1
		) VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,

	"writeImageToken": `
		INSERT INTO image_tokens (token, SHA1, expires) VALUES
			($1, $2, $3)`,

	"writeOP": `
		INSERT INTO threads (
			board, log, id, postCtr, imageCtr, replyTime, bumpTime, subject
		) VALUES
			($1, $2, $3, $4, $5, $6, $7, $8)`,

	"newPostID": `SELECT nextval('post_id')`,

	"getAllBoard": `
		SELECT t.board, t.id, t.postCtr, t.imageCtr, t.replyTime, t.bumpTime,
				t.subject, p.spoiler, p.time, p.name, p.trip, p.auth,
				p.imageName,
				(SELECT array_length(t.log, 1)) AS logCtr,
				i.*
			FROM threads AS t
			INNER JOIN posts AS p
				ON t.id = p.id AND (p.deleted is NULL OR p.deleted = 'false')
			LEFT OUTER JOIN images AS i
				ON p.SHA1 = i.SHA1
			ORDER BY bumpTime DESC`,

	"getBoard": `
		SELECT t.board, t.id, t.postCtr, t.imageCtr, t.replyTime, t.bumpTime,
				t.subject, p.spoiler, p.time, p.name, p.trip, p.auth,
				p.imageName,
				(SELECT array_length(t.log, 1)) AS logCtr,
				i.*
			FROM threads AS t
			INNER JOIN posts AS p
				ON t.id = p.id AND (deleted IS NULL OR deleted = 'false')
			LEFT OUTER JOIN images AS i
				ON p.SHA1 = i.SHA1
			WHERE t.board = $1
			ORDER BY bumpTime DESC`,

	"getPost": `
		SELECT op, board, editing, banned, spoiler, id, time, body, name, trip,
				auth, links, backlinks, commands, imageName, images.*
			FROM posts
			LEFT OUTER JOIN images
				ON posts.SHA1 = images.SHA1
			WHERE id = $1 AND (deleted IS NULL OR deleted = 'false')`,

	"getThread": `
		SELECT board, postCtr, imageCtr, replyTime, bumpTime, subject,
				(SELECT array_length(log, 1)) AS logCtr
			FROM threads
			WHERE id = $1`,

	"getThreadPost": `
		SELECT editing, banned, spoiler, id, time, body, name, trip, auth,
				links, backlinks, commands, imageName, images.*
			FROM posts
			LEFT OUTER JOIN images
				ON posts.SHA1 = images.SHA1
			WHERE id = $1 AND (deleted IS NULL OR deleted = 'false')`,

	"getFullThread": `
		SELECT editing, banned, spoiler, id, time, body, name, trip, auth,
				links, backlinks, commands, imageName, images.*
			FROM posts
			LEFT OUTER JOIN images
				ON posts.SHA1 = images.SHA1
			WHERE op = $1
				AND id != $1
				AND (deleted IS NULL OR deleted = 'false')
			ORDER BY id ASC`,

	"getLastN": `
		WITH t AS (
			SELECT editing, banned, spoiler, id, time, body, name, trip, auth,
				links, backlinks, commands, imageName, images.*
			FROM posts
			LEFT OUTER JOIN images
				ON posts.SHA1 = images.SHA1
			WHERE op = $1
				AND id != $1
				AND (deleted IS NULL OR deleted = 'false')
			ORDER BY id DESC
			LIMIT $2
		)
		SELECT * FROM t ORDER BY id ASC`,

	"threadCounter": `SELECT array_length(log, 1) FROM threads WHERE id = $1`,

	"boardCounter": `SELECT ctr FROM boards WHERE id = $1`,

	"validateOP": `SELECT true FROM threads WHERE id = $1 AND board = $2`,

	"getImage": `SELECT * FROM images WHERE SHA1 = $1`,

	"useImageToken": `
		DELETE FROM image_tokens
			WHERE token = $1
			RETURNING SHA1`,

	"closeExpiredOpenPosts": `
		UPDATE posts
			SET editing = false
			WHERE editing = true AND time < EXTRACT(EPOCH FROM now()) - 1800
			RETURNING id, op`,

	"updateLog": `
		UPDATE threads
			SET log = array_append(log, $2)
			WHERE id = $1`,

	"isLoggedIn": `SELECT true FROM sessions WHERE account = $1 AND token = $2`,

	"getPassword": `SELECT password FROM accounts WHERE id = $1`,
}
