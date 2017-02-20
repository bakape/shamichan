insert into threads (
	board, log, id, postCtr, imageCtr, replyTime, bumpTime, subject
)
	values ($1, $2, $3, $4, $5, $6, $7, $8)
