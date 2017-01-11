package db

// func TestValidateOp(t *testing.T) {
// 	assertTableClear(t, "threads")
// 	assertInsert(t, "threads", common.DatabaseThread{
// 		ID:    1,
// 		Board: "a",
// 	})

// 	samples := [...]struct {
// 		id      uint64
// 		board   string
// 		isValid bool
// 	}{
// 		{1, "a", true},
// 		{15, "a", false},
// 	}

// 	for i := range samples {
// 		s := samples[i]
// 		t.Run("", func(t *testing.T) {
// 			t.Parallel()
// 			valid, err := ValidateOP(s.id, s.board)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
// 			if valid != s.isValid {
// 				t.Fatal("unexpected result")
// 			}
// 		})
// 	}
// }
