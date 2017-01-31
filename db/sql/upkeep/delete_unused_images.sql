delete from images
	where (
		select count(*) from posts
			where SHA1 = images.SHA1
	) = 0
	returning SHA1, fileType, thumbType
