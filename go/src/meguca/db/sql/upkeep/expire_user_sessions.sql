delete from sessions
	where expires < now() at time zone 'utc'
