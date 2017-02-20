package main

import (
	"fmt"
	"os"

	"meguca/common"
	"meguca/config"
	"meguca/db"
	"meguca/util"
	"github.com/dancannon/gorethink"
)

var (
	rSession *gorethink.Session

	// For verifying foreign keys
	accounts, images, boards []string
)

type links map[uint64]struct {
	OP uint64
}

func main() {
	var err error
	rSession, err = gorethink.Connect(gorethink.ConnectOpts{
		Address: "localhost:28015",
	})
	if err != nil {
		panic(err)
	}
	rSession.Use("meguca")

	db.IsTest = true // Prevent cleanup tasks from running
	err = db.LoadDB()
	if err != nil {
		panic(err)
	}

	err = util.Waterfall(
		copyMeta, copyAccounts, copyImages, copyBoards, copyThreads,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func copyMeta() (err error) {
	fmt.Println("copying metadata")

	var info struct {
		PostCtr uint64
		Pyu     uint
	}
	err = One(gorethink.Table("main").Get("info"), &info)
	if err != nil {
		return
	}

	err = db.SetPostCounter(info.PostCtr)
	if err != nil {
		return
	}
	err = db.SetPyu(info.Pyu)
	if err != nil {
		return
	}

	var conf config.Configs
	err = One(gorethink.Table("main").Get("config"), &conf)
	if err != nil {
		return
	}
	return db.WriteConfigs(conf)
}

func copyAccounts() (err error) {
	const h = "copying accounts"
	fmt.Print(h)
	defer fmt.Print("\n")

	var acs []struct {
		ID       string
		Password []byte
	}
	err = All(gorethink.Table("accounts"), &acs)
	if err != nil {
		return
	}

	// Admin account is created on init
	err = db.ClearTables("accounts")
	if err != nil {
		return
	}
	for i, ac := range acs {
		accounts = append(accounts, ac.ID)
		err = db.RegisterAccount(ac.ID, ac.Password)
		if err != nil {
			return
		}
		printProgress(h, i, len(acs))
	}

	return nil
}

func copyImages() (err error) {
	const h = "copying images"
	fmt.Print(h)
	defer fmt.Print("\n")

	var imgs []common.ImageCommon
	err = All(gorethink.Table("images"), &imgs)
	if err != nil {
		return
	}

	tx, err := db.StartTransaction()
	if err != nil {
		return
	}
	defer db.RollbackOnError(tx, &err)

	for i, img := range imgs {
		images = append(images, img.SHA1)
		err = db.WriteImage(tx, img)
		if err != nil {
			return
		}
		printProgress(h, i, len(imgs))
	}

	return tx.Commit()
}

func copyBoards() (err error) {
	const h = "copying boards"
	fmt.Print(h)
	defer fmt.Print("\n")

	var bs []struct {
		db.BoardConfigs
		Staff map[string][]string
	}
	err = All(gorethink.Table("boards"), &bs)
	if err != nil {
		return
	}

	tx, err := db.StartTransaction()
	if err != nil {
		return
	}
	defer db.RollbackOnError(tx, &err)

	for i, b := range bs {
		boards = append(boards, b.ID)
		err = db.WriteBoard(tx, b.BoardConfigs)
		if err != nil {
			return
		}

		owner := b.Staff["owners"][0]
		ownerExists := false
		for _, id := range accounts {
			if id == owner {
				ownerExists = true
				break
			}
		}
		if !ownerExists {
			printProgress(h, i, len(bs))
			continue
		}

		err = db.WriteStaff(tx, b.ID, b.Staff)
		if err != nil {
			return
		}
		printProgress(h, i, len(bs))
	}

	return tx.Commit()
}

func copyThreads() (err error) {
	for _, b := range boards {
		fmt.Printf("copying /%s/ threads\n", b)

		var threads []uint64
		q := gorethink.Table("threads").GetAllByIndex("board", b).Field("id")
		err = All(q, &threads)
		if err != nil {
			return
		}
		for _, id := range threads {
			err = copyThread(id)
			if err != nil {
				return
			}
		}
	}
	return nil
}

func copyThread(id uint64) (err error) {
	h := fmt.Sprintf("  copying thread %d", id)
	fmt.Print(h)
	defer fmt.Print("\n")

	var thread db.Thread
	err = One(gorethink.Table("threads").Get(id), &thread)
	if err != nil {
		return
	}

	var posts []struct {
		db.Post
		Links, Backlinks links
	}
	q := gorethink.Table("posts").GetAllByIndex("op", id).OrderBy("id")
	err = All(q, &posts)
	if err != nil {
		return
	}

	// Assert thread has an OP
	if len(posts) == 0 || posts[0].ID != id {
		return
	}

	// Compute fields from posts and verify images
	thread.PostCtr = 0
	thread.ImageCtr = 0
	thread.Log = [][]byte{}
	for i, p := range posts {
		thread.ReplyTime = p.Time
		if i < 1000 {
			thread.BumpTime = p.Time
		}
		thread.PostCtr++

		// Assert image foreign key
		if p.Image != nil {
			matched := false
			for _, id := range images {
				if p.Image.SHA1 == id {
					matched = true
					break
				}
			}
			if !matched {
				p.Image = nil
			}
		}

		if p.Image != nil {
			thread.ImageCtr++
		}

		p.Post.Links = convertLinks(p.Links)
		p.Post.Backlinks = convertLinks(p.Backlinks)
		p.IP = ""
		p.Password = nil
		p.Board = thread.Board
		posts[i] = p
	}

	tx, err := db.StartTransaction()
	if err != nil {
		return
	}
	defer db.RollbackOnError(tx, &err)

	err = db.WriteThread(tx, thread, posts[0].Post)
	if err != nil {
		return
	}
	printProgress(h, 0, len(posts))

	if len(posts) < 2 {
		return tx.Commit()
	}
	for i := 1; i < len(posts); i++ {
		err = db.WritePost(tx, posts[i].Post)
		if err != nil {
			return
		}
		printProgress(h, i, len(posts))
	}

	return tx.Commit()
}

// Convert links to new format
func convertLinks(links links) (conv [][2]uint64) {
	for id, l := range links {
		conv = append(conv, [2]uint64{id, l.OP})
	}
	return
}
