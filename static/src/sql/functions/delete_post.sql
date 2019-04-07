create or replace function delete_post(id bigint, account text)
returns void as $$
declare
	target_board text;
begin
	-- Get post board
	select post_board(p.id) into target_board
		from posts p
		where p.id = delete_post.id
			and not is_deleted(p.id);

	-- No match
	if target_board is null then
		return;
	end if;

	-- Assert user can delete posts on board, if not already done
	perform assert_can_perform(account, target_board, 1::smallint);

	-- Delete post
	insert into mod_log (type, board, post_id, "by")
		values (2, target_board, id, account);
end;
$$ language plpgsql;
