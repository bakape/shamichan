create or replace function after_post_update()
returns trigger
language plpgsql
as $$
begin
	if (old.image is null and new.image is not null)
		or (old.open and not new.open)
	then
		call bump_thread(id => new.thread, page => new.page);
	end if;
end;
$$;
