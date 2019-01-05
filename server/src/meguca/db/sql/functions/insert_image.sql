-- Inserts image into existing post and return image json
create function insert_image(post_id bigint, token char(86), name varchar(200),
	spoiler bool)
returns jsonb as $$
declare
	sha1 char(40);
	data jsonb;
begin
	update posts
		set sha1 = use_image_token(token),
			imageName = name,
			spoiler = spoiler
		where id = post_id
		returning posts.sha1 into sha1;

	select to_jsonb(*) into data
		from images i
		where i.sha1 = sha1;
	perform jsonb_insert(data, '{id}', to_jsonb(post_id));
	perform jsonb_insert(data, '{spoiler}', to_jsonb(spoiler));
	perform jsonb_insert(data, '{name}', to_jsonb(name));
	return data;
end;
$$ language plpgsql;
