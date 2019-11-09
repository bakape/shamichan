delete from images as i
where
	(
		(
			select count(*)
			from posts p
			where p.sha1 = i.sha1
		)
		+ (
			select count(*)
			from image_tokens it
			where it.sha1 = i.sha1
		)
	) = 0
returning i.SHA1, i.file_type, i.thumb_type
