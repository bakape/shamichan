-- Encode post row to json
create or replace function encode_post(p posts)
returns jsonb
language sql stable parallel safe strict
as $$
declare
	data jsonb;
	tmp jsonb;

	img images%rowtype;
begin
	data = jsonb_build_object(
		'id', p.id,
		'time', p.time,
		'body', p.body,
		'page', p.page
	);
	if p.trip != '' then
		data = data || jsonb_build_object('trip', p.trip);
	end if;
	if p.name != '' then
		data = data || jsonb_build_object('name', p.name);
	end if;
	if p.commands is not null then
		data = data || jsonb_build_object('commands', p.commands);
	end if;
	if p.sage then
		data = data || jsonb_build_object('sage', p.sage);
	end if;
	if p.flag != '' then
		data = data || jsonb_build_object('flag', p.flag);
	end if;
	if p.editing then
		data = data || jsonb_build_object('editing', p.editing);
	end if;
	if p.auth != 0 then
		data = data || jsonb_build_object('auth', p.auth);
	end if;

	-- Links
	select get_links(p.id) into tmp;
	if tmp is not null then
		data = jsonb_set(data, '{links}', tmp);
	end if;

	-- Backlinks
	select into tmp
		jsonb_object_agg(
			l.source,
			jsonb_build_object(
				'op', lp.op,
				'board', t.board
			)
		)
		from links l
		join posts lp on lp.id = l.source
		join threads t on lp.op = t.id
		where l.target = p.id;
	if tmp is not null then
		data = jsonb_set(data, '{backlinks}', tmp);
	end if;

	-- Image
	if p.sha1 is not null then
		select i.* into img
			from images i
			where i.sha1 = p.sha1;

		tmp = jsonb_build_object(
			'name', p.imageName,
			'file_type', img.file_type,
			'thumb_type', img.thumb_type,
			'dims', img.dims,
			'size', img.size,
			'md5', img.md5,
			'sha1', img.sha1
		);

		if p.spoiler then
			tmp = tmp || jsonb_build_object('spoiler', true);
		end if;
		if img.audio then
			tmp = tmp || jsonb_build_object('audio', true);
		end if;
		if img.video then
			tmp = tmp || jsonb_build_object('video', true);
		end if;
		if img.length != 0 then
			tmp = tmp || jsonb_build_object('length', img.length);
		end if;
		if img.title != '' then
			tmp = tmp || jsonb_build_object('title', img.title);
		end if;
		if img.artist != '' then
			tmp = tmp || jsonb_build_object('artist', img.artist);
		end if;

		data = jsonb_set(data, '{image}', tmp);
	end if;

	-- Moderation
	if p.moderated then
		select get_post_moderation(p.id) into tmp;
		if tmp is not null then
			data = jsonb_set(data, '{moderation}', tmp);
		end if;
	end if;

	return data;
end;
$$;
