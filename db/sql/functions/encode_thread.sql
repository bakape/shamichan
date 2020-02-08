-- Encode thread column into struct
create or replace function encode_thread(t threads, page int)
returns jsonb
language plpgsql stable parallel safe strict
as $$
begin
	return jsonb_build_object(
		'id', t.id,
		'board', t.board,
		'post_count', post_count(t.id),
		'image_count', (
			select count(*) into image_count
			from posts p
			where p.thread = t.id and p.image is not null
		),
		'page', page,
		'update_time', t.update_time,
		'bump_time', t.bump_time,
		'subject', t.subject
	);
end;
$$;
