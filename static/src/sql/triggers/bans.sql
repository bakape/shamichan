create or replace function notify_bans_updated()
returns trigger
language plpgsql
as $$
begin
	perform pg_notify('bans.updated', '');
	return null;
end;
$$;

