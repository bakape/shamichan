update posts
	set ip = null
	where time < extract(epoch from now() at time zone 'utc' - interval '7 days')
