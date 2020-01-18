-- Encode post row to json
create or replace function encode_post(p posts)
returns jsonb
language sql stable parallel safe strict
as $$
declare
	data jsonb;

	img images;
begin
	data = jsonb_build_object(
		'id', p.id,
		'thread', p.thread,
		'page', p.page,

		'created_on', p.created_on,
		'open', p.open,

		'sage', p.sage,
		'name', p.name,
		'trip', p.trip,
		'flag', p.flag,

		'body', p.body
	);

	if p.image is not null then
		select i.* into img
			from images i
			where i.sha1 = p.image;

		data = data || jsonb_build_object(
			'image', to_jsonb(img) || jsonb_build_object(
				'name',
			)
		)
	end if;

	return data;
end;
$$;
