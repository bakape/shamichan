update threads
	set
		replyTime = case when $2
			then floor(extract(epoch from now()))
			else replyTime
		end,
		postCtr = case when $2
			then postCtr + 1
			else postCtr
		end,
		bumpTime = case when $3
			then
				case when postCtr <= 1000
					then floor(extract(epoch from now()))
					else bumpTime
				end
			else bumpTime
		end,
		imageCtr = case when $4
			then imageCtr + 1
			else imageCtr
		end
	where id = $1
