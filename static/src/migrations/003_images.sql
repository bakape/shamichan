
create type file_type as enum (
	'JPEG',
	'PNG',
	'GIF',
	'WEBM',
	'PDF',
	'SVG',
	'MP4',
	'MP3',
	'OGG',
	'ZIP',
	'7z',
	'TGZ',
	'TXZ',
	'FLAC',
	'NO_FILE',
	'TXT',
	'WEBP',
	'RAR',
	'CBZ',
	'CBR'
);

create table images (
	sha1 bytea primary key check (octet_length(sha1) = 20),
	md5 bytea not null check(octet_length(md5) = 16),

	audio bool not null,
	video bool not null,
	file_type file_type not null,
	thumb_type file_type not null,
	dims smallint[] not null check(array_length(dims, 1) = 4),
	size int not null check (size > 0),
	title varchar(200) not null,
	artist varchar(100) not null
);
