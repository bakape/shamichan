select board, postCtr, imageCtr, replyTime, bumpTime, subject
	from threads
	where id = $1
