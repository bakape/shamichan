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
	page bigint check (page >= 0),

	created_on timestamptz_auto_now,
	open bool not null default true,
	auth_key auth_key,

	body jsonb,

	image bytea references images,
	image_name varchar(200) not null default '',
	image_spoilered bool not null default false
)
inherits (post_common);

create index posts_thread_idx on posts (thread);
create index posts_page_idx on posts (page);
create index posts_open_idx on posts (open);
create index posts_created_on_idx on posts (created_on);
create index posts_auth_key_idx on posts (auth_key);
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
begin
	-- +1, because new post is not inserted yet
	new.page =  (post_count(new.op) + 1) / 100;

	-- TODO: Bump thread
	-- perform bump_thread(new.op, bump_time => not new.sage, page => new.page);

	-- TODO: Moderation
	-- -- Delete post, if IP blacklisted
	-- select b.by into to_delete_by
	-- 	from bans b
	-- 	-- Can't use post_board(), because not inserted yet
	-- 	where
	-- 		board = (
	-- 			select t.board
	-- 			from threads t
	-- 			where t.id = new.op
	-- 		)
	-- 		and b.ip = new.ip
	-- 		and b.type = 'shadow'
	-- 		and b.expires > now() at time zone 'UTC';
	-- if to_delete_by is not null then
	-- 	-- Will fail otherwise, because key is not written to table yet
	-- 	set constraints post_moderation_post_id_fkey deferred;
	-- 	insert into post_moderation (post_id, type, "by")
	-- 		values (new.id, 2, to_delete_by);
	-- 	new.moderated = true;
	-- end if;

	return new;
end;
$$;

create or replace function merge_jsonb_obj_st(in out s jsonb, item jsonb)
language plpgsql stable parallel safe strict
as $$
begin
	s = s || item;
end;
$$;

create aggregate merge_jsonb_obj(item jsonb) (
	sfunc = merge_jsonb_obj_st,
	stype = jsonb,
	initcond = '{}'
);
