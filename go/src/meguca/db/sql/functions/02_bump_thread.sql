create or replace function bump_thread(
	id bigint,
	bump bool,
	image bool
) returns void as $$
	update threads
		set
			replyTime = floor(extract(epoch from now())),
			postCtr =  postCtr + 1,
			bumpTime = case when bump
				then
					case when postCtr <= 5000
						then floor(extract(epoch from now()))
						else bumpTime
					end
				else bumpTime
			end,
			imageCtr = case when image
				then imageCtr + 1
				else imageCtr
			end
		where id = bump_thread.id;
$$ language sql;
