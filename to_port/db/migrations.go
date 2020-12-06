package db

// // Run migrations, till the DB version matches the code version
// func runMigrations() (err error) {
// 	var migrations []string
// 	err = static.Walk(
// 		"/migrations",
// 		func(path string, info os.FileInfo, wlkErr error) (err error) {
// 			if wlkErr != nil {
// 				return wlkErr
// 			}
// 			if info.IsDir() {
// 				return
// 			}
// 			migrations = append(migrations, path)
// 			return
// 		},
// 	)
// 	if err != nil {
// 		return
// 	}
// 	target := len(migrations)

// 	b := context.Background()

// 	// Init main table, if not done yet
// 	var exists bool
// 	err = db.QueryRow(b, `select to_regclass('main') is not null`).Scan(&exists)
// 	if err != nil {
// 		return
// 	}
// 	if !exists {
// 		_, err = db.Exec(
// 			b,
// 			`create table main (
// 				key text primary key,
// 				val jsonb not null
// 			)`,
// 		)
// 		if err != nil {
// 			return
// 		}
// 		_, err = db.Exec(
// 			b,
// 			`insert into main (key, val)
// 			values ('version', '0');`,
// 		)
// 		if err != nil {
// 			return
// 		}
// 	}

// 	for {
// 		var (
// 			current int
// 			done    bool
// 		)
// 		err = InTransaction(context.Background(), func(tx pgx.Tx) (err error) {
// 			var _current string

// 			// Lock version column to ensure no migrations from other processes
// 			// happen concurrently
// 			err = tx.
// 				QueryRow(
// 					b,
// 					`select val
// 					from main
// 					where key = 'version'
// 					for update`,
// 				).
// 				Scan(&_current)
// 			if err != nil {
// 				return
// 			}
// 			current, err = strconv.Atoi(_current)
// 			if err != nil {
// 				return
// 			}
// 			if current == target {
// 				done = true
// 				return
// 			}
// 			if current > target {
// 				log.Fatal("database version ahead of codebase")
// 			}

// 			if !common.IsTest {
// 				log.Infof("upgrading database to version %d", current+1)
// 			}

// 			buf, err := static.ReadFile(migrations[current])
// 			if err != nil {
// 				return
// 			}
// 			_, err = tx.Exec(b, string(buf))
// 			if err != nil {
// 				return
// 			}

// 			// Write new version number
// 			_, err = tx.Exec(
// 				b,
// 				`update main
// 				set val = $1
// 				where key = 'version'`,
// 				strconv.Itoa(current+1),
// 			)
// 			return
// 		})
// 		if err != nil {
// 			return fmt.Errorf(
// 				"migration error: %s: %s",
// 				migrations[current],
// 				err,
// 			)
// 		}
// 		if done {
// 			return
// 		}
// 	}
// }
