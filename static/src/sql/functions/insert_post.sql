create or replace function insert_post(
	op bigint,
	id bigint,
	body text,
	flag text,
	name text,
	trip text,
	auth smallint,
	sage bool,
	password bytea,
	ip inet
)
returns jsonb
language strict
as $$
declare
	creation_time bigint;
begin
	-- Thread OPs will have a predefined id, but all others need one generated
	if id = 0 then
		id = nextval('post_id');
	end if;

	insert into posts (
			op,
			id,
			body,
			flag,
			name,
			trip,
			auth,,
			sage,
			password,
			ip
		)
		values (
			op,
			id,
			body,
			flag,
			name,
			trip,
			auth,,
			sage,
			password,
			ip
		)
		returning time into creation_time;

	return jsonb_build_object(
		'id', id,
		'time', creation_time,
		'moderation', get_post_moderation(id)
	);
end;
$$;
