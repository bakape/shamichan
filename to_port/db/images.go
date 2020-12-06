package db

// // SpoilerImage spoilers an already allocated image
// func SpoilerImage(ctx context.Context, id uint64) error {
// 	_, err := db.Exec(
// 		ctx,
// 		`update posts
// 		set image_spoilered = true
// 		where id = $1 and image is not null`,
// 		id,
// 	)
// 	return err
// }
