create domain auth_key as bytea check (octet_length(value) = 64);
create domain timestamptz_auto_now as timestamptz not null default now();

create sequence post_id_seq as bigint;

create table post_common (
	id
		bigint
		not null
		default nextval('post_id_seq'::regclass)
		check (id > 0)
);

create table threads (
	primary key (id),
	subject varchar(100) not null,
	bump_time timestamptz_auto_now,
	tags varchar(20)[] not null check (array_length(tags, 1) between 1 and 3)
)
inherits (post_common);

create index threads_tags_idx on threads using gin (tags);

create table posts (
	primary key (id),
	thread bigint not null references threads on delete cascade,
	created_on timestamptz_auto_now,
	auth_key auth_key not null,
	body bytea
)
inherits (post_common);

create index posts_thread_idx on posts (thread);
