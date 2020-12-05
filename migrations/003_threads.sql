create domain timestamptz_auto_now as timestamptz not null default now();

create sequence post_id_seq as bigint;

create table post_common (
	id
		bigint
		not null
		default nextval('post_id_seq'::regclass)
		check (id > 0),
	created_on timestamptz_auto_now
);

create table threads (
	primary key (id),
	subject varchar(100) not null,
	bumped_on timestamptz_auto_now,
	tags varchar(20)[] not null check (array_length(tags, 1) between 1 and 3)
)
inherits (post_common);

create index threads_tags_idx on threads using gin (tags);

create table public_keys (
	id bigserial primary key,
	public_id uuid unique not null,
	public_key bytea unique not null check (octet_length(public_key) <= 1024)
);

create table posts (
	primary key (id),
	thread bigint not null references threads on delete cascade,
	page int not null default 0 check (page >= 0),

	open bool not null default true,

	sage bool not null default false,
	name varchar(100),
	trip varchar(100),
	flag char(2),

	public_key bigint references public_keys,

	body jsonb not null,

	image bytea references images,
	image_name varchar(200) not null default '',
	image_spoilered bool not null default false
)
inherits (post_common);

create index posts_thread_idx on posts (thread);
create index posts_page_idx on posts (page);
create index posts_open_idx on posts (open);
create index posts_created_on_idx on posts (created_on);
create index posts_public_key_idx on posts (public_key);
create index posts_image_idx on posts (image);

create or replace function post_count(thread bigint)
returns bigint
language sql stable parallel safe strict
as $$
	select count(*)
	from posts
	where posts.thread = post_count.thread;
$$;

create or replace function before_posts_insert()
returns trigger
language plpgsql
as $$
declare
	posts_in_thread bigint;
begin
	posts_in_thread = post_count(new.thread);
	new.page = case
		when posts_in_thread = 0 then 0
		else posts_in_thread / 100
	end;

	if not new.sage then
		call bump_thread(new.thread);
	end if;

	return new;
end;
$$;

create trigger before_posts_insert
before insert on posts
for each row execute procedure before_posts_insert();

-- Bump a thread to top of index
create or replace procedure bump_thread(id bigint)
language sql
as $$
	update threads as t
	set bumped_on = now()
	where t.id = bump_thread.id;
$$;

