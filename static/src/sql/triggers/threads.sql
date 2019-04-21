create or replace function after_threads_insert()
returns trigger as $$
begin
	perform bump_thread(new.id);

	-- Init Russian roulette
	insert into roulette (id, scount, rcount) values (new.id, 6, 0);

	return null;
end;
$$ language plpgsql;

create or replace function after_threads_update()
returns trigger as $$
begin
	-- Prevent infinite recursion on timestamp updates
	if new.replyTime != old.replyTime then
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
