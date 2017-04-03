package db

import "meguca/common"

// AppendBody appends a character to a post body
func AppendBody(id, op uint64, char rune) error {
	msg, err := common.EncodeMessage(
		common.MessageAppend,
		[2]uint64{id, uint64(char)},
	)
	if err != nil {
		return err
	}
	return execPrepared("append_body", id, op, string(char), msg)
}

// Writes new backlinks to other posts
func insertBackinks(id, op uint64, links [][2]uint64) (err error) {
	// Deduplicate
	dedupped := make(map[[2]uint64]struct{}, len(links))
	for _, l := range links {
		dedupped[l] = struct{}{}
	}

	// Most often this loop will iterate only once, so no need to think heavily
	// on optimizations
	for l := range dedupped {
		var msg []byte
		msg, err = common.EncodeMessage(
			common.MessageBacklink,
			[3]uint64{l[0], id, op},
		)
		if err != nil {
			return
		}
		err = execPrepared(
			"insert_backlink",
			l[0], l[1], msg, linkRow{{id, op}},
		)
		if err != nil {
			return
		}
	}

	return
}

// Backspace removes one character from the end of the post body
func Backspace(id, op uint64) error {
	msg, err := common.EncodeMessage(common.MessageBackspace, id)
	if err != nil {
		return err
	}
	return execPrepared("backspace", id, op, msg)
}

// ClosePost closes an open post and commits any links, backlinks and hash
// commands
func ClosePost(id, op uint64, links [][2]uint64, com []common.Command) (
	err error,
) {
	err = execPrepared(
		"close_post",
		id, op, linkRow(links), commandRow(com),
	)
	if err != nil {
		return
	}

	if links != nil {
		err = insertBackinks(id, op, links)
		if err != nil {
			return
		}
	}

	return err
}

// SplicePost splices the text body of a post. For less load on the DB, supply
// the entire new body as `body`.
func SplicePost(id, op uint64, msg []byte, body string) error {
	return execPrepared("splice_body", id, op, body, msg)
}
