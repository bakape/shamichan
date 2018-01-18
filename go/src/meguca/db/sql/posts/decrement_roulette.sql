update main
	set val = (val::smallint - 1)::text
	where id = 'roulette'
	returning val::smallint + 1
