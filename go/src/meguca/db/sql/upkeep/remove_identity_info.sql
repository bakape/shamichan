update posts
	set ip = null
	where time < extract(epoch from now() - interval '7 days')
