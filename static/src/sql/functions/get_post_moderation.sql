-- Retrieve moderation actions performed on post
create or replace function get_post_moderation(id bigint)
returns jsonb
language sql stable parallel safe strict
as $$
declare
	data jsonb;
begin
	select jsonb_agg(
		jsonb_build_object(
			'type', pm.type,
			'length', pm.length,
			'by', pm.by,
			'data', pm.data
		)
	)
	from post_moderation pm
	where pm.post_id = get_post_moderation.id;
end;
$$;
