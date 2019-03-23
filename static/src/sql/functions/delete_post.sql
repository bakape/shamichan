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

	-- Assert user can delete posts on board
	perform assert_can_perform(account, target_board, 1::smallint);

	-- Delete post
	perform delete_post(id, account, target_board, 0);
end;
$$ language plpgsql;

-- Runs the post deletion operation. Authorization checks should already be
-- completed prior to calling this.
create or replace function delete_post(id bigint, account text, board text,
	length bigint)
returns void as $$
begin
	insert into mod_log (type, board, post_id, "by", length)
		values (2, board, id, account, length);
end;
$$ language plpgsql;
