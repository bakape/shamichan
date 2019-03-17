create or replace function delete_posts_by_ip(id bigint, account text)
returns void as $$
declare
	target_board text;
	target_ip inet;
	id bigint;
begin
	-- Get post board and IP
	select t.board, p.ip into target_board, target_ip
		from posts p
		join threads t on (p.op = t.id)
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
				join threads t on (p.op = t.id)
				where p.ip = target_ip
					and t.board = target_board
					-- Ensure not already deleted
					and not is_deleted(p.id))
	loop
		perform delete_post(id, account, target_board);
	end loop;
end;
$$ language plpgsql;
