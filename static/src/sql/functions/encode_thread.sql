-- Encode thread column into struct
create or replace function encode_thread(t threads, page int)
returns jsonb as $$
declare
	data jsonb;
	image_count bigint;
begin
	select count(*) into image_count
		from posts p
		where p.op = t.id and p.sha1 is not null;

	data = jsonb_build_object(
		'board', t.board,
		'post_count', post_count(t.id),
		'image_count', image_count,
		'page', page,
		'update_time', t.update_time,
		'bump_time', t.bump_time,
		'subject', t.subject
	);

	if t.locked then
		data = data || jsonb_build_object('locked', true);
	end if;
	if t.sticky then
		data = data || jsonb_build_object('sticky', true);
	end if;

	return data;
end;
$$ language plpgsql;
