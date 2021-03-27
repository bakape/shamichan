-- Inserts image into existing post and return image json
create function insert_image(post_id bigint, token char(86), name varchar(200),
	spoiler bool)
returns jsonb as $$
declare
	image_id char(40);
	data jsonb;
begin
	update posts
		set sha1 = use_image_token(insert_image.token),
			imageName = insert_image.name,
			spoiler = insert_image.spoiler
		where id = post_id
		returning posts.sha1 into image_id;
	if image_id is null then
		raise exception 'post not found';
	end if;

	select to_jsonb(i) into data
		from images i
		where i.sha1 = image_id;
	return data || jsonb_build_object(
		'id', post_id,
		'spoiler', spoiler,
		'name', name);
end;
$$ language plpgsql;