create or replace function delete_posts(id bigint, account text, by_ip boolean)
returns void as $$
declare
	target_board text;
	post_id bigint;
	target_ip inet;
	ids bigint[];
begin
	-- Get post board and IP
	select post_board(p.id), p.ip into target_board, target_ip
		from posts p
		where p.id = delete_posts.id;

	-- Assert user can delete posts on board
	perform assert_can_perform(account, target_board, 1::smallint);

	if by_ip and target_ip is not null then
		ids := array(select p.id
						from posts p
						where p.ip = target_ip
							and post_board(p.id) = target_board
							-- Ensure not already deleted
							and not is_deleted(p.id));
	else
		-- Still need to check if the targeted post has been deleted
		ids := array(select p.id
						from posts p
						where p.id = delete_posts.id
							and not is_deleted(p.id));
	end if;

	-- Delete the posts
	foreach post_id in array ids loop
		insert into mod_log (type, board, post_id, "by")
			values (2, target_board, post_id, account);
	end loop;
end;
$$ language plpgsql;
