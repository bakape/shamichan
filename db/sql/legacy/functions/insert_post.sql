create or replace function insert_post(
	op bigint,
	id bigint,
	body text,
	flag text,
	name text,
	trip text,
	sage bool,
	user_id uuid,
	ip inet
)
returns jsonb
language strict
as $$
declare
	creation_time bigint;
	page bigint;
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
			sage,
			user_id,
			ip
		)
		values (
			op,
			id,
			body,
			flag,
			name,
			trip,
			sage,
			user_id,
			ip
		)
		returning time, page into creation_time, page;

	return jsonb_build_object(
		'id', id,
		'time', creation_time,
		'page', page
	);
end;
$$;
