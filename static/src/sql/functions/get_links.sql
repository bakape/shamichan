create or replace function get_links(id bigint)
returns jsonb as $$
	select jsonb_object_agg(
		l.target,
		jsonb_build_object(
			'op', lp.op,
			'board', t.board
		)
	)
	from links l
	join posts lp on lp.id = l.target
	join threads t on lp.op = t.id
	where l.source = get_links.id;
$$ language sql;
