select close_post(
	$1::bigint,
	$2::bigint,
	$3::varchar(2000),
	$4::bigint[][2],
	$5::json[]
)
