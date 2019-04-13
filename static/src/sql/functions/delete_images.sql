create or replace function delete_images(ids bigint[], account text)
returns void as $$
declare
	board text;
	checked_boards jsonb = '{}';
	post_id bigint;
begin
	foreach post_id in array ids loop
		select post_board(p.id) into board
			from posts p
			where p.id = post_id
				and p.sha1 is not null;
		if board is null then
			continue;
		end if;

		if not checked_boards?board then
			perform assert_can_perform(account, board, 1::smallint);
			checked_boards = checked_boards || jsonb_build_object(board, true);
		end if;

		insert into mod_log (type, board, post_id, "by")
			values (3, board, post_id, account);
	end loop;
end;
$$ language plpgsql;
