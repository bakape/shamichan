create or replace function notify_thread_post_count()
returns trigger as $$
begin
	perform pg_notify('new_post_in_thread', new.op || ',' || (
		select count(*)
		from posts
		where posts.op = new.op));
	return null;
end;
$$ language plpgsql;
