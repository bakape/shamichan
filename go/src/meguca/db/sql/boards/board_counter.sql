select max(replyTime) + count(*) from threads
	where board = $1
