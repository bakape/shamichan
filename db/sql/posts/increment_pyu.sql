update main
	set val = (val::bigint + 1)::text
	where id = 'pyu'
	returning val::bigint
