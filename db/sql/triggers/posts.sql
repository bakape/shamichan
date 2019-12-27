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
