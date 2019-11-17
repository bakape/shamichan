create or replace function after_threads_insert()
returns trigger
language plpgsql
as $$
begin
	perform bump_thread(new.id);
	return null;
end;
$$;

create or replace function after_threads_update()
returns trigger
language plpgsql
as $$
begin
	-- Prevent infinite recursion on timestamp updates
	if new.update_time != old.update_time then
		perform bump_thread(new.id, board => new.board);
	end if;
	return null;
end;
$$;

create or replace function after_threads_delete()
returns trigger
language plpgsql
as $$
begin
	perform pg_notify('thread.deleted', old.id);
	return null;
end;
$$;
