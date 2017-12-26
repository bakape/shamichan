insert into posts (
		editing, spoiler, id, board, op, time, body, flag, posterID,
		name, trip, auth, password, ip, SHA1, imageName, links, commands, sage
	)
	values ($1,	$2,	$3,	$4,	$5,	$6,	$7,	$8,	$9,	$10, $11, $12, $13,	$14, $15, $16, $17, $18, $19)
	returning bump_thread($5, true, not $19, $15 is not null)
