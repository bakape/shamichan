update posts
	set ip = null, password = null
	where time < extract(epoch from now() - interval '7 days')
