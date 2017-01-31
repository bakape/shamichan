delete from sessions
	where expires < now()
