package db

// func assertDeleted(t *testing.T, q string, del bool) {
// 	t.Helper()

// 	q = fmt.Sprintf(`select exists (select 1 %s)`, q)
// 	var exists bool
// 	err := db.QueryRow(context.Background(), q).Scan(&exists)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	deleted := !exists
// 	if deleted != del {
// 		test.LogUnexpected(t, del, deleted)
// 	}
// }

// func assertThreadDeleted(t *testing.T, id uint64, del bool) {
// 	t.Helper()

// 	q := fmt.Sprintf(`from threads where id = '%d'`, id)
// 	assertDeleted(t, q, del)
// }
