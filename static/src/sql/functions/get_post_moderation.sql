-- Retrieve moderation actions performed on post
create or replace function get_post_moderation(id bigint)
returns jsonb as $$
declare
	data jsonb;
begin
	select into data
		jsonb_agg(
			jsonb_build_object(
				'type', pm.type,
				'length', pm.length,
				'by', pm.by,
				'data', pm.data
			)
		)
		from post_moderation pm
		where pm.post_id = get_post_moderation.id;
	return data;
end;
$$ language plpgsql;
