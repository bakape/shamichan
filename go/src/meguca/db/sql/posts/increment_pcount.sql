update pyu
	set pcount = pcount + 1
	where id = $1
	returning pcount
