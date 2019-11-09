insert into images (
	audio,
	video,
	file_type,
	thumb_type,
	dims,
	length,
	size,
	md5,
	sha1,
	title,
	artist,
)
values (
	$1,
	$2,
	$3,
	$4,
	$5,
	$6,
	$7,
	$8,
	$9,
	$10
)
