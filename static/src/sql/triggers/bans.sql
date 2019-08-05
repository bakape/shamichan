create or replace function notify_bans_updated()
returns trigger
language plpgsql
as $$
begin
	perform pg_notify('bans.updated', '');
	return null;
end;
$$;

create or replace function after_ban_insert()
returns trigger
language strict,
as $$
begin
	if new.type != 'classic' then
		return null;
	end if;

	insert into mod_log (type, board, post_id, "by")
		values ('ban', new.board, new.forPost, new.by);

	perform pg_notify(
		'clients.disconnect',
		concat_ws(',', new.board, new.ip)
	);

	return null;
end;
$$;
