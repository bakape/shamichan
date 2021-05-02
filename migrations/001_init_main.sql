create table main (
	key text primary key,
	val jsonb not null
);

create or replace function notify_config_updates()
returns trigger
language plpgsql
as $$
begin
	if new.key = 'config' then
		perform pg_notify('config.updated', '');
	end if;
	return new;
end;
$$;

create trigger notify_config_updates
after insert or update on main
for each row
execute function notify_config_updates();
