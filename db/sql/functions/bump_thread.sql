-- Bump a thread and propagate it has been updated
--
-- op: id of thread being bumped
-- bump_time: also update the thread's bumped_on
-- page: page of the thread to bump. Used for cache invalidation.
create or replace procedure bump_thread(
	id bigint,
	bump_time bool = false,
	page int = -2
)
language plpgsql strict
as $$
begin
	if bump_thread.bump_time then
		update threads as t
			set bumped_on = now()
			where t.id = bump_thread.id;
	end if;

	perform pg_notify('thread.updated',	concat_ws(',', id, page));
end;
$$;
