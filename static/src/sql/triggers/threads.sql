create or replace function after_threads_insert()
returns trigger as $$
begin
	perform bump_thread(new.id);
	return null;
end;
$$ language plpgsql;

create or replace function after_threads_update()
returns trigger as $$
begin
	-- Prevent infinite recursion on timestamp updates
	if new.update_time != old.update_time then
		perform bump_thread(new.id);
	end if;
	return null;
end;
$$ language plpgsql;

create or replace function after_threads_delete()
returns trigger as $$
begin
	perform pg_notify('thread_deleted', old.board || ',' || old.id);
	return null;
end;
$$ language plpgsql;
