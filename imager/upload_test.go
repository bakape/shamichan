package imager

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/test"
	"github.com/bakape/meguca/test/test_db"
	"github.com/jackc/pgx/v4"
)

func TestUpload(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name, fileName, downloadName string
		img                          common.ImageCommon
	}{
		{
			name:         "MP3 no cover",
			fileName:     "sample.mp3",
			downloadName: "sample",
			img: common.ImageCommon{
				Audio:     true,
				FileType:  common.MP3,
				ThumbType: common.NoFile,
				Duration:  1,
				Size:      0x782c,
			},
		},
		{
			name:         "already processed file",
			fileName:     "sample.mp3",
			downloadName: "sample",
			img: common.ImageCommon{
				Audio:     true,
				FileType:  common.MP3,
				ThumbType: common.NoFile,
				Duration:  1,
				Size:      0x782c,
			},
		},
		{
			name:         "MP3 with cover",
			fileName:     "with_cover.mp3",
			downloadName: "with_cover",
			img: common.ImageCommon{
				Audio:       true,
				Video:       true,
				FileType:    common.MP3,
				ThumbType:   common.WEBP,
				Duration:    1,
				Size:        0x0a8b82,
				Width:       0x0500,
				Height:      0x02d0,
				ThumbWidth:  0x96,
				ThumbHeight: 0x54,
			},
		},
		{
			name:         "ZIP",
			fileName:     "sample.zip",
			downloadName: "sample",
			img: common.ImageCommon{
				FileType:  common.ZIP,
				ThumbType: common.NoFile,
				Size:      0x096941,
			},
		},
		{
			name:         "CBZ",
			fileName:     "manga.zip",
			downloadName: "manga",
			img: common.ImageCommon{
				FileType:    common.CBZ,
				ThumbType:   common.WEBP,
				Size:        0x0968a9,
				ThumbWidth:  0x96,
				ThumbHeight: 0x54,
			},
		},
		{
			name:         "RAR",
			fileName:     "sample.rar",
			downloadName: "sample",
			img: common.ImageCommon{
				FileType:  common.RAR,
				ThumbType: common.NoFile,
				Size:      0x096bb2,
			},
		},
		{
			name:         "CBR",
			fileName:     "manga.rar",
			downloadName: "manga",
			img: common.ImageCommon{
				FileType:    common.CBR,
				ThumbType:   common.WEBP,
				Size:        0x096b18,
				ThumbWidth:  0x96,
				ThumbHeight: 0x54,
			},
		},
		{
			name:         "7Z",
			fileName:     "sample.7z",
			downloadName: "sample",
			img: common.ImageCommon{
				FileType:  common.SevenZip,
				ThumbType: common.NoFile,
				Size:      0x0181,
			},
		},
		{
			name:         "tar.gz",
			fileName:     "sample.tar.gz",
			downloadName: "sample",
			img: common.ImageCommon{
				FileType:  common.TGZ,
				ThumbType: common.NoFile,
				Size:      0x096a28,
			},
		},
		{
			name:         "tar.xz",
			fileName:     "sample.tar.xz",
			downloadName: "sample",
			img: common.ImageCommon{
				FileType:  common.TXZ,
				ThumbType: common.NoFile,
				Size:      0x096b6c,
			},
		},
		{
			name:         "PDF",
			fileName:     "sample.pdf",
			downloadName: "sample",
			img: common.ImageCommon{
				FileType:  common.PDF,
				ThumbType: common.NoFile,
				Size:      0x39ed,
			},
		},
		{
			name:         "big file path",
			fileName:     "testdata.zip",
			downloadName: "testdata",
			img: common.ImageCommon{
				FileType:  common.ZIP,
				ThumbType: common.NoFile,
				Size:      0xe64fb9,
			},
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			thread, user := test_db.InsertSampleThread(t)

			body := new(bytes.Buffer)
			w := multipart.NewWriter(body)
			fw, err := w.CreateFormFile("image", c.fileName)
			if err != nil {
				t.Fatal(err)
			}
			f := test.OpenSample(t, c.fileName)
			_, err = io.Copy(fw, f)
			if err != nil {
				t.Fatal(err)
			}
			err = w.Close()
			if err != nil {
				t.Fatal(err)
			}

			var sha1Hash [20]byte
			_, err = hashFile(sha1Hash[:], f, sha1.New())
			if err != nil {
				t.Fatal(err)
			}
			var md5Hash [16]byte
			_, err = hashFile(md5Hash[:], f, md5.New())
			if err != nil {
				t.Fatal(err)
			}
			err = f.Close()
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest("POST", "/", body)
			req.Header.Set("Authorization", "Bearer "+user.String())
			req.Header.Set("Content-Length", strconv.Itoa(body.Len()))
			req.Header.Set("Content-Type", w.FormDataContentType())
			rec := httptest.NewRecorder()
			NewImageUpload(rec, req)
			if rec.Code != 200 {
				t.Fatalf("failed thumbnailing: %s", rec.Body.String())
			}

			var img common.ImageCommon
			err = db.InTransaction(context.Background(), func(tx pgx.Tx) (err error) {
				img, err = db.GetImage(context.Background(), tx, sha1Hash)
				return
			})
			if err != nil {
				t.Fatal(err)
			}
			c.img.SHA1 = sha1Hash
			c.img.MD5 = md5Hash
			test.AssertEquals(t, img, c.img)

			var post struct {
				Image *common.Image
			}
			buf, err := db.GetPost(context.Background(), thread)
			if err != nil {
				t.Fatal(err)
			}
			err = json.Unmarshal(buf, &post)
			if err != nil {
				t.Fatal(err)
			}
			test.AssertEquals(t, post.Image, &common.Image{
				Name:        c.downloadName,
				ImageCommon: c.img,
			})
		})
	}
}
