package db

import "../common"

// UpdateLog writes to a thread's replication log. Only used in tests.
func UpdateLog(id uint64, msg []byte) error {
	_, err := db.Exec(`select update_log($1, $2)`, id, msg)
	return err
}

// AppendBody appends a character to a post body
func AppendBody(id, op uint64, char rune, body string) error {
	msg, err := common.EncodeMessage(
		common.MessageAppend,
		[2]uint64{id, uint64(char)},
	)
	if err != nil {
		return err
	}
	bodyModCh <- bodyModRequest{
		id:   id,
		op:   op,
		msg:  msg,
		body: body,
	}
	return nil
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
func Backspace(id, op uint64, body string) error {
	msg, err := common.EncodeMessage(common.MessageBackspace, id)
	if err != nil {
		return err
	}
	bodyModCh <- bodyModRequest{
		id:   id,
		op:   op,
		msg:  msg,
		body: body,
	}
	return nil
}

// ClosePost closes an open post and commits any links, backlinks and hash
// commands
func ClosePost(id, op uint64, links [][2]uint64, com []common.Command) (
	err error,
) {
	msg, err := common.EncodeMessage(common.MessageClosePost, struct {
		ID       uint64           `json:"id"`
		Links    [][2]uint64      `json:"links,omitempty"`
		Commands []common.Command `json:"commands,omitempty"`
	}{
		ID:       id,
		Links:    links,
		Commands: com,
	})

	err = execPrepared(
		"close_post",
		id, op, msg, linkRow(links), commandRow(com),
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
func SplicePost(id, op uint64, msg []byte, body string) {
	bodyModCh <- bodyModRequest{
		id:   id,
		op:   op,
		msg:  msg,
		body: body,
	}
}
