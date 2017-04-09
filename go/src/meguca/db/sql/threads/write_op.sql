insert into threads (
	board, id, postCtr, imageCtr, replyTime, bumpTime, subject
)
	values ($1, $2, $3, $4, $5, $6, $7)
