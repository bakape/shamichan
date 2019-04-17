create or replace function delete_posts(ids bigint[], account text)
returns void as $$
declare
	board text;
	checked_boards jsonb = '{}';
	post_id bigint;
begin
	foreach post_id in array ids loop
		-- Get post board
		select post_board(p.id) into board
			from posts p
			where p.id = post_id
				and not is_deleted(p.id);

		-- No match
		if board is null then
			continue;
		end if;

		-- Assert user can delete posts on board, if not already checked
		if not checked_boards?board then
			perform assert_can_perform(account, board, 1::smallint);
			checked_boards = checked_boards || jsonb_build_object(board, true);
		end if;

		-- Delete post
		insert into mod_log (type, board, post_id, "by")
			values (2, board, post_id, account);
	end loop;
end;
$$ language plpgsql;
