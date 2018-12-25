create or replace function notify_thread_deleted()
returns trigger as $$
begin
	perform pg_notify('thread_deleted', old.board || ',' || old.id);
	return null;
end;
$$ language plpgsql;
