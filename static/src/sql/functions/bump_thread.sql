-- Bump a thread and propagate it has been updated
-- op: id of thread being bumped
-- bump_time: also update the thread's bump_time
-- board: passes the board of the thread. If null, the board is automatically
-- 	detected computed.
-- page: page of the thread to bump. Used for cache invalidation.
-- 	Default bumps all pages.
create or replace procedure bump_thread(
	op bigint,
	bump_time bool = false,
	board text = null,
	page int = -2
)
language plpgsql
as $$
declare
	now_unix bigint = extract(epoch from now());
begin
	update threads
		set update_time = now_unix
		where id = op;
	if bump_thread.bump_time and post_count(bump_thread.op) < 1000 then
		update threads
			set bump_time = now_unix
			where id = bump_thread.op;
	end if;

	if board is null then
		select t.board into board
			from threads t
			where t.id = op;
		if board is null then
			raise exception 'thread does not exist: %', op;
		end if;
	end if;
	perform pg_notify('thread.updated',	concat_ws(',', board, op, page));
end;
$$;
