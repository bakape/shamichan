select board, postCtr, imageCtr, replyTime, bumpTime, subject,
		(select array_length(log, 1)) as logCtr
	from threads
	where id = $1
