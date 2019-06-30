create or replace function before_posts_insert()
returns trigger as $$
declare
	to_delete_by text;
	post_counter bigint;
begin
	-- +1, because new post is not inserted yet
	post_counter = post_count(new.op) + 1;
	new.page = post_counter / 100;

	perform bump_thread(new.op, bump_time => not new.sage, page => new.page);
	perform pg_notify('thread.new_post', new.op || ',' || post_counter);

	-- Delete post, if IP blacklisted
	select b.by into to_delete_by
		from bans b
		-- Can't use post_board(), because not inserted yet
		where board = (select t.board
						from threads t
						where t.id = new.op)
			and b.ip = new.ip
			and b.type = 'shadow'
			and b.expires > now() at time zone 'UTC';
	if to_delete_by is not null then
		-- Will fail otherwise, because key is not written to table yet
		set constraints post_moderation_post_id_fkey deferred;
		insert into post_moderation (post_id, type, "by")
			values (new.id, 2, to_delete_by);
		new.moderated = true;
	end if;

	return new;
end;
$$ language plpgsql;

create or replace function after_posts_update()
returns trigger as $$
begin
	if new.editing != old.editing then
		perform bump_thread(
			new.op,
			bump_time => not new.sage,
			page => new.page
		);
	end if;
	return null;
end;
$$ language plpgsql;
