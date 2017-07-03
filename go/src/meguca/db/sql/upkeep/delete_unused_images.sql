delete from images
	where (
		(select count(*) from posts where SHA1 = images.SHA1)
		+ (select count(*) from image_tokens where SHA1 = images.SHA1)
	) = 0
	returning SHA1, fileType, thumbType
