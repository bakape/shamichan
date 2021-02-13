
-- Encode thread column into struct
create or replace function encode(t threads, page bigint, page_count bigint)
returns jsonb
language plpgsql stable parallel safe strict
as $$
begin
	return jsonb_build_object(
		'id', t.id,
		'page', page,
		'page_count', page_count,

		'subject', t.subject,
		'tags', t.tags,

		'bumped_on', to_unix(t.bumped_on),
		'created_on', to_unix(t.created_on),
		'post_count', post_count(t.id),
		'image_count', (
			select count(*)
			from posts p
			where p.thread = t.id and p.image is not null
		)
	);
end;
$$;

drop function encode(threads, bigint);

-- Get thread JSON
-- page: thread page to fetch.
-- 	If -1, fetches last page.
-- 	If -5, fetches last 5 posts.
create or replace function get_thread(id bigint, page bigint)
returns jsonb
language plpgsql stable parallel safe strict
as $$
declare
	max_page bigint;
	thread threads%rowtype;

	data jsonb;
	posts jsonb;
begin
	select max(p.page) into max_page
		from posts p
		where p.thread = get_thread.id;
	if max_page is null or page > max_page then
		return null;
	end if;
	if page = -1 then
		page = max_page;
	end if;

	select encode(t, page, max_page + 1) into data
		from threads t
		where t.id = get_thread.id;
	if data is null then
		return null;
	end if;

	case page
	when -5 then
		data = data || '{"page":0}';
		select into posts
			json_object_agg(pp.id, encode(pp))
			from (
				select *
				from posts p
				where p.id = get_thread.id

				union all

				select *
				from (
					select *
					from posts p
					where p.thread = get_thread.id
						and p.id != get_thread.id
					order by p.id desc
					limit 5
				) _
			) pp;
	else
		if page < 0 then
			raise exception 'invalid page number %', page;
		end if;

		select into posts
			json_object_agg(p.id, encode(p))
			from posts p
			where (p.thread = get_thread.id and p.page = get_thread.page)
				or p.id = get_thread.id;
	end case;
	data = jsonb_set(data, '{posts}', posts);

	return data;
end;
$$;
