create or replace function delete_post(id bigint, account text)
returns void as $$
declare
	target_board text;
begin
	-- Get post board
	select t.board into target_board
		from posts p
		join threads t on (p.op = t.id)
		where p.id = delete_post.id
			and not is_deleted(p.id);

	-- No match
	if target_board is null then
		return;
	end if;

	-- Assert user can delete posts on board
	perform assert_can_perform(account, target_board, 1::smallint);

	-- Delete post
	perform delete_post(id, account, target_board);
end;
$$ language plpgsql;

-- Runs the post deletion operation. Authorization checks should already be
-- completed prior to calling this.
create or replace function delete_post(id bigint, account text, board text)
returns void as $$
begin
	insert into mod_log (type, board, post_id, "by")
		values (2, board, id, account);
end;
$$ language plpgsql;
