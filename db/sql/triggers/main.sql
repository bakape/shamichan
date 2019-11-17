create or replace function after_main_update()
returns trigger
language plpgsql strict
as $$
begin
	if new.key = 'config' then
		perform pg_notify('configs.updated', '');
	end if;
end;
$$;
