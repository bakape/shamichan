create function notify_thread_deleted()
returns trigger as $$
begin
	perform pg_notify('thread_deleted', old.id::text);
	return null;
end;
$$ language plpgsql;
