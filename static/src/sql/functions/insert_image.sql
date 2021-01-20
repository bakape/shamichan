-- Inserts image into existing post and return image json
create function insert_image(post_id bigint, token char(86), name varchar(200),
	spoiler bool, mask bool)
returns jsonb as $$
declare
	image_id char(40);
	data jsonb;
	masked_name varchar(200);
begin
	image_id = use_image_token(insert_image.token);
	if image_id is null then
		raise exception 'post not found';
	end if;

	if insert_image.mask then
		masked_name = image_id;
	else
		masked_name = insert_image.name;
	end if;

	update posts
		set sha1 = image_id,
			imageName = masked_name,
			spoiler = insert_image.spoiler
		where id = post_id;

	select to_jsonb(i) into data
		from images i
		where i.sha1 = image_id;
	return data || jsonb_build_object(
		'id', post_id,
		'spoiler', spoiler,
		'name', masked_name);
end;
$$ language plpgsql;
