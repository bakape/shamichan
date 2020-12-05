create or replace function after_post_update()
returns trigger
language plpgsql
as $$
begin
	if not new.sage
		and (
			(old.image is null and not new.image is null)
			or (old.open and not new.open)
		)
	then
		call bump_thread(new.thread);
	end if;
end;
$$;


-- Encode post row to json
create or replace function encode(p posts)
returns jsonb
language plpgsql stable parallel safe strict
as $$
declare
	data jsonb;
	img images;
begin
	data = jsonb_build_object(
		'id', p.id,
		'thread', p.thread,
		'page', p.page,

		'created_on', to_unix(p.created_on),
		'open', p.open,

		'sage', p.sage,
		'name', p.name,
		'trip', p.trip,
		'flag', p.flag,

		'body', p.body,
		'image', null
	);

	if p.image is not null then
		select i.* into img
			from images i
			where i.sha1 = p.image;

		data = data || jsonb_build_object(
			'image', jsonb_build_object(
				'name', p.image_name,
				'spoilered', p.image_spoilered,

				'sha1', encode(img.sha1, 'hex'),
				'md5', encode(img.md5, 'hex'),

				'audio', img.audio,
				'video', img.video,

				'file_type', img.file_type,
				'thumb_type', img.thumb_type,

				'width', img.width,
				'height', img.height,
				'thumb_width', img.thumb_width,
				'thumb_height', img.thumb_height,

				'size', img.size,
				'duration', img.duration,

				'title', img.title,
				'artist', img.artist
			)
		);
	end if;

	return data;
end;
$$;
