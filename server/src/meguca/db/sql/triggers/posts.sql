create or replace function on_posts_insert()
returns trigger as $$
begin
	perform bump_thread(new.op, not new.sage);
	perform pg_notify('new_post_in_thread',
		new.op || ',' || post_count(new.op));
	return null;
end;
$$ language plpgsql;

create or replace function on_posts_update()
returns trigger as $$
begin
	perform bump_thread(new.op);
	return null;
end;
$$ language plpgsql;
