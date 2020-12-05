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
	'7Z',
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

create domain uint15 as smallint check (value >= 0);

create table images (
	id bigserial primary key,

	sha1 bytea not null check (octet_length(sha1) = 20),
	md5 bytea not null check (octet_length(md5) = 16),

	audio bool not null,
	video bool not null,

	file_type file_type not null,
	thumb_type file_type not null,

	width uint15 not null,
	height uint15 not null,
	thumb_width uint15 not null,
	thumb_height uint15 not null,

	size bigint not null check (size > 0),
	duration int not null check (duration >= 0),

	title varchar(200),
	artist varchar(100)
);
