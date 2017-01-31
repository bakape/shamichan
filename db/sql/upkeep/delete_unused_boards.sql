delete from boards
	where
		created < $1
		and (select coalesce(max(replyTime), 0) from threads
				where board = boards.id
			) < extract(epoch from $1)
	returning pg_notify('board_updated', boards.id)
