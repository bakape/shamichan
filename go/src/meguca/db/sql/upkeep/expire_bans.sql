delete from bans
	where expires < now() at time zone 'utc'
