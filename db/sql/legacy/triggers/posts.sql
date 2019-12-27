create or replace function after_posts_update()
returns trigger
language plpgsql
as $$
begin
	if old.editing and not new.editing then
		perform bump_thread(
			new.op,
			bump_time => not new.sage,
			page => new.page
		);
	end if;
	return null;
end;
$$;
