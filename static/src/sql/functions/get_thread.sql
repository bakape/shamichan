-- Get thread JSON
-- page: thread page to fetch.
-- 	If -1, fetches last page.
-- 	If -5, fetches last 5 posts.
create or replace function get_thread(id bigint, page int)
returns jsonb as $$
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

	if page != -5 then
		select into posts
			jsonb_agg(encode_post(p) order by p.id)
			from posts p
			where (p.op = get_thread.id and p.page = get_thread.page)
				or p.id = get_thread.id;
	else
		select into posts
			jsonb_agg(encode_post(pp))
			from (
				select p.*
				from posts p
				where p.op = get_thread.id or p.id = get_thread.id
				order by p.id
				limit 6
			) pp;
	end if;
	data = jsonb_set(data, '{posts}', posts);

	return data;
end;
$$ language plpgsql;
