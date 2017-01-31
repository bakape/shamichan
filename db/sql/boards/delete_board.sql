delete from boards
	where id = $1
	returning pg_notify('board_updated', $1)
