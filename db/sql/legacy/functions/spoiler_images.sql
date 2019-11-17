create or replace function spoiler_images(ids bigint[], account text)
returns void
language plpgsql strict
as $$
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
			perform call bump_thread(account, board, 1::smallint);
			checked_boards = checked_boards || jsonb_build_object(board, true);
		end if;

		update posts as p
			set spoiler = true
			where p.id = post_id;
		insert into mod_log (type, board, post_id, "by")
			values ('spoiler_image', board, post_id, account);
	end loop;
end;
$$;
