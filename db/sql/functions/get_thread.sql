-- Get thread JSON
-- page: thread page to fetch.
-- 	If -1, fetches last page.
-- 	If -5, fetches last 5 posts.
--	If -6, fetches only the OP.
create or replace function get_thread(id bigint, page int)
returns jsonb
language plpgsql stable parallel safe strict
as $$
declare
	max_page int;
	thread threads%rowtype;

	data jsonb;
	posts jsonb;
begin
	select max(p.page) into max_page
		from posts p
		where p.op = get_thread.id;
	if max_page is null or page > max_page then
		return null;
	end if;
	if page = -1 then
		page = max_page;
	end if;

	select encode_thread(t, page) into data
		from threads t
		where t.id = get_thread.id;
	if data is null then
		return null;
	end if;

	case page
	when -5 then
		data = data - 'page';
		select into posts
			jsonb_agg(encode_post(pp) order by pp.id)
			from (
				select *
				from posts p
				where p.id = get_thread.id
				union all
				select *
				from (
					select *
					from posts p
					where p.op = get_thread.id and p.id != get_thread.id
					order by p.id desc
					limit 5
				) _
			) pp;
	when -6 then
		data = data - 'page';
		select into posts
			jsonb_agg(encode_post(p))
			from posts p
			where p.id = get_thread.id;
	else
		if page < 0 then
			raise exception 'invalid page number %', page;
		end if;

		select into posts
			jsonb_agg(encode_post(p) order by p.id)
			from posts p
			where (p.op = get_thread.id and p.page = get_thread.page)
				or p.id = get_thread.id;
	end case;
	data = jsonb_set(data, '{posts}', posts);

	return data;
end;
$$;
