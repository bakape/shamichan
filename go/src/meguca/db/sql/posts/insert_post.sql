insert into posts (
		spoiler, id, board, op, time, body, flag, posterID,
		name, trip, auth, ip, SHA1, imageName, links, commands, sage
	)
	values ($1,	$2,	$3,	$4,	$5,	$6,	$7,	$8,	$9,	$10, $11, $12, $13,	$14, $15, $16, $17)
	returning bump_thread($4, true, not $17, $13 is not null)
