create or replace function encode_mod_log(m mod_log)
language sql immutable parallel safe strict
returns jsonb as $$
	select jsonb_build_object(
		'type', m.type,
		'length', m.length,
		'by', m.by,
		'data', m.data
	);
$$;
