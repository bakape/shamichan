-- length: keep deleting posts of this IP for duration in seconds
create or replace function delete_posts_by_ip(id bigint, account text,
	length bigint)
returns void as $$
declare
	target_board text;
	target_ip inet;
	id bigint;
begin
	-- Get post board and IP
	select post_board(p.id), p.ip into target_board, target_ip
		from posts p
		where p.id = delete_posts_by_ip.id;

	-- Post gone or already past 7 days old
	if target_ip is null then
		return;
	end if;

	-- Assert user can delete posts on board
	perform assert_can_perform(account, target_board, 1::smallint);

	-- Delete the posts
	for id in (select p.id
				from posts p
				where p.ip = target_ip
					and post_board(p.id) = target_board
					-- Ensure not already deleted
					and not is_deleted(p.id))
	loop
		perform delete_post(id, account, target_board, length);
	end loop;

	-- Keep deleting posts till this expires
	if length > 0 then
		insert into continuous_deletions (ip, board, "by", till)
			values (target_ip, target_board, account,
				(now() + make_interval(secs := length)) at time zone 'utc');
	end if;
end;
$$ language plpgsql;
