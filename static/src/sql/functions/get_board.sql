-- Fetch a page of a board index
create or replace function get_board(board varchar, page int)
returns jsonb
language plpgsql stable parallel safe strict
as $$
declare
	data jsonb;
	max_page int;
begin
	select count(*) / 10 into max_page
		from threads t
		where t.board = get_board.board;
	if max_page is null or page > max_page then
		return null;
	end if;

	select into data
		jsonb_agg(
			get_thread(id, -5) - 'page'
			order by sticky desc, bump_time desc
		)
		from (
			select *
			from threads t
			where t.board = get_board.board
			order by t.sticky desc, t.bump_time desc
			limit 10
			offset 10 * get_board.page
		) tt;
	if data is null then
		return null;
	end if;
	return jsonb_build_object(
		'page', page,
		'pages', max_page + 1,
		'threads', data
	);
end;
$$;

-- Fetch a page of the /all/ board index
create or replace function get_all_board(page int)
returns jsonb
language plpgsql stable parallel safe strict
as $$
declare
	data jsonb;
	max_page int;
begin
	select count(*) / 10 into max_page from threads t;
	if max_page is null or page > max_page then
		return null;
	end if;

	select into data
		jsonb_agg(
			get_thread(id, -5) - 'page'
			order by bump_time desc
		)
		from (
			select *
			from threads t
			order by t.bump_time desc
			limit 10
			offset 10 * get_all_board.page
		) tt;
	if data is null then
		return null;
	end if;
	return jsonb_build_object(
		'page', page,
		'pages', max_page + 1,
		'threads', data
	);
end;
$$;
