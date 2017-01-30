update posts
	set editing = false
	where editing = true
		and time < floor(extract(epoch from now())) - 1800
	returning id, op
