-- Encode post row to json
create or replace function encode_post(p posts)
returns jsonb as $$
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

		data = data || jsonb_build_object('image', tmp);
	end if;

	if p.moderated then
		select into tmp
			jsonb_agg(
				jsonb_build_object(
					'type', pm.type,
					'length', pm.length,
					'by', pm.by,
					'data', pm.data
				)
			)
			from post_moderation pm
			where pm.post_id = p.id;
		if tmp is not null then
			data = data || jsonb_build_object('moderation', tmp);
		end if;
	end if;

	return data;
end;
$$ language plpgsql;
