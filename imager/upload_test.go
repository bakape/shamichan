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
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/test"
	"github.com/bakape/meguca/test/test_db"
	"github.com/jackc/pgx/v4"
)

type uploadCase struct {
	name, fileName, downloadName string
	img                          common.ImageCommon
	code                         int
	err                          string
}

func TestUpload(t *testing.T) {
	t.Parallel()

	var (
		invalidTitle  = "ti?"
		invalidArtist = "art\x01?"
		title         = "Puella Magi Madoka Magica Part III - Rebellion"
	)

	cases := [...]uploadCase{
		{
			name:         "MP3 no cover",
			fileName:     "sample.mp3",
			downloadName: "sample",
			code:         200,
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
			code:         200,
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
			code:         200,
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
			code:         200,
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
			code:         200,
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
			code:         200,
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
			code:         200,
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
			code:         200,
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
			code:         200,
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
			code:         200,
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
			code:         200,
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
			code:         200,
			img: common.ImageCommon{
				FileType:  common.ZIP,
				ThumbType: common.NoFile,
				Size:      0xe64fb9,
			},
		},
		{
			name:         "JPEG",
			fileName:     "sample.jpg",
			downloadName: "sample",
			code:         200,
			img: common.ImageCommon{
				FileType:    common.JPEG,
				ThumbType:   common.WEBP,
				Width:       0x043c,
				Height:      0x0371,
				ThumbWidth:  0x96,
				ThumbHeight: 0x79,
				Size:        0x0496f8,
			},
		},
		{
			name:     "too tall",
			fileName: "too_tall.jpg",
			code:     400,
			err:      "invalid input: invalid image: image too tall\n",
		},
		{
			name:     "too wide", // No such thing
			fileName: "too_wide.jpg",
			code:     400,
			err:      "invalid input: invalid image: image too wide\n",
		},
		{
			name:         "MP3 + invalid UTF-8 metainformation",
			fileName:     "invalid_utf8.mp3",
			downloadName: "invalid_utf8",
			code:         200,
			img: common.ImageCommon{
				Audio:     true,
				FileType:  common.MP3,
				ThumbType: common.NoFile,
				Duration:  1,
				Size:      0x7c89,
				Title:     &invalidTitle,
				Artist:    &invalidArtist,
			},
		},
		{
			name:         "MP4",
			fileName:     "sample.mp4",
			downloadName: "sample",
			code:         200,
			img: common.ImageCommon{
				Audio:       true,
				Video:       true,
				FileType:    common.MP4,
				ThumbType:   common.WEBP,
				Duration:    0x0d,
				Width:       0x0500,
				Height:      0x02d0,
				ThumbWidth:  0x96,
				ThumbHeight: 0x54,
				Size:        0x2029c2,
			},
		},
		{
			name:         "MP4 + AAC",
			fileName:     "aac.mp4",
			downloadName: "aac",
			code:         200,
			img: common.ImageCommon{
				Audio:     true,
				FileType:  common.MP4,
				ThumbType: common.NoFile,
				Duration:  0x0d,
				Size:      0x09e387,
			},
		},
		{
			name:         "MP4 + .H264",
			fileName:     "h264.mp4",
			downloadName: "h264",
			code:         200,
			img: common.ImageCommon{
				Video:       true,
				FileType:    common.MP4,
				ThumbType:   common.WEBP,
				Duration:    0x0d,
				Width:       0x0500,
				Height:      0x02d0,
				ThumbWidth:  0x96,
				ThumbHeight: 0x54,
				Size:        0x1638c7,
			},
		},
		{
			name:         "MP4 + .H264 + MP3 ",
			fileName:     "mp3_h264.mp4",
			downloadName: "mp3_h264",
			code:         200,
			img: common.ImageCommon{
				Audio:       true,
				Video:       true,
				FileType:    common.MP4,
				ThumbType:   common.WEBP,
				Duration:    0x0d,
				Width:       0x0500,
				Height:      0x02d0,
				ThumbWidth:  0x96,
				ThumbHeight: 0x54,
				Size:        0x199dac,
			},
		},
		{
			name:         "MP4 + MP3 ",
			fileName:     "mp3.mp4",
			downloadName: "mp3",
			code:         200,
			img: common.ImageCommon{
				Audio:     true,
				FileType:  common.MP4,
				ThumbType: common.NoFile,
				Duration:  0x0d,
				Size:      0x0350e7,
			},
		},
		{
			name:         "MP4 + cover",
			fileName:     "with_cover.mp4",
			downloadName: "with_cover",
			code:         200,
			img: common.ImageCommon{
				Audio:       true,
				Video:       true,
				FileType:    common.MP4,
				ThumbType:   common.WEBP,
				Duration:    0x0d,
				Width:       0x043c,
				Height:      0x0371,
				ThumbWidth:  0x96,
				ThumbHeight: 0x79,
				Size:        0x07ebff,
			},
		},
		{
			name:         "OGG",
			fileName:     "sample.ogg",
			downloadName: "sample",
			code:         200,
			img: common.ImageCommon{
				Audio:       true,
				Video:       true,
				FileType:    common.OGG,
				ThumbType:   common.WEBP,
				Duration:    0x05,
				Width:       0x0230,
				Height:      0x0140,
				ThumbWidth:  0x96,
				ThumbHeight: 0x55,
				Size:        0x06affc,
			},
		},
		{
			name:         "OGG - audio",
			fileName:     "no_audio.ogg",
			downloadName: "no_audio",
			code:         200,
			img: common.ImageCommon{
				Video:       true,
				FileType:    common.OGG,
				ThumbType:   common.WEBP,
				Duration:    0x05,
				Width:       0x0230,
				Height:      0x0140,
				ThumbWidth:  0x96,
				ThumbHeight: 0x55,
				Size:        0x059e96,
			},
		},
		{
			name:         "OGG + Opus + Theora",
			fileName:     "opus_theora.ogg",
			downloadName: "opus_theora",
			code:         200,
			img: common.ImageCommon{
				Audio:       true,
				Video:       true,
				FileType:    common.OGG,
				ThumbType:   common.WEBP,
				Duration:    0x05,
				Width:       0x0230,
				Height:      0x0140,
				ThumbWidth:  0x96,
				ThumbHeight: 0x55,
				Size:        0x064dc9,
			},
		},
		{
			name:         "OGG - video",
			fileName:     "no_video.ogg",
			downloadName: "no_video",
			code:         200,
			img: common.ImageCommon{
				Audio:     true,
				FileType:  common.OGG,
				ThumbType: common.NoFile,
				Duration:  0x05,
				Size:      0xb2a3,
			},
		},
		{
			name:         "OGG - video + cover",
			fileName:     "with_cover.ogg",
			downloadName: "with_cover",
			code:         200,
			img: common.ImageCommon{
				Audio:     true,
				FileType:  common.OGG,
				ThumbType: common.NoFile,
				Duration:  0x05,
				Size:      0x06d8e4,
			},
		},
		{
			name:         "OGG - video + Opus ",
			fileName:     "opus.ogg",
			downloadName: "opus",
			code:         200,
			img: common.ImageCommon{
				Audio:     true,
				FileType:  common.OGG,
				ThumbType: common.NoFile,
				Duration:  0x05,
				Size:      0xafcc,
			},
		},
		{
			name:         "PNG",
			fileName:     "sample.png",
			downloadName: "sample",
			code:         200,
			img: common.ImageCommon{
				FileType:    common.PNG,
				ThumbType:   common.WEBP,
				Width:       0x0500,
				Height:      0x02d0,
				ThumbWidth:  0x96,
				ThumbHeight: 0x54,
				Size:        0x09af2e,
			},
		},
		{
			name:         "APNG",
			fileName:     "sample.apng",
			downloadName: "sample",
			code:         200,
			img: common.ImageCommon{
				FileType:    common.PNG,
				ThumbType:   common.WEBP,
				Width:       0x64,
				Height:      0x64,
				ThumbWidth:  0x64,
				ThumbHeight: 0x64,
				Size:        0x010017,
			},
		},
		{
			name:         "GIF",
			fileName:     "sample.gif",
			downloadName: "sample",
			code:         200,
			img: common.ImageCommon{
				FileType:    common.GIF,
				ThumbType:   common.WEBP,
				Width:       0x0248,
				Height:      0x02d0,
				ThumbWidth:  0x79,
				ThumbHeight: 0x96,
				Size:        0x0367bb,
			},
		},
		{
			name:         "too small to thumbnail",
			fileName:     "too_small.png",
			downloadName: "too_small",
			code:         200,
			img: common.ImageCommon{
				FileType:    common.PNG,
				ThumbType:   common.WEBP,
				Width:       0x79,
				Height:      0x96,
				ThumbWidth:  0x79,
				ThumbHeight: 0x96,
				Size:        0x5cb2,
			},
		},
		{
			name:         "WEBM",
			fileName:     "sample.webm",
			downloadName: "sample",
			code:         200,
			img: common.ImageCommon{
				Audio:       true,
				Video:       true,
				FileType:    common.WEBM,
				ThumbType:   common.WEBP,
				Width:       0x0500,
				Height:      0x02d0,
				ThumbWidth:  0x96,
				ThumbHeight: 0x54,
				Duration:    0x01,
				Size:        0x026910,
				Title:       &title,
			},
		},
		{
			name:         "WEBP",
			fileName:     "sample.webp",
			downloadName: "sample",
			code:         200,
			img: common.ImageCommon{
				FileType:    common.WEBP,
				ThumbType:   common.WEBP,
				Width:       0x0500,
				Height:      0x02d0,
				ThumbWidth:  0x96,
				ThumbHeight: 0x54,
				Size:        0x530e,
			},
		},
		{
			name:         "TXT",
			fileName:     "sample.txt",
			downloadName: "sample",
			code:         200,
			img: common.ImageCommon{
				FileType:  common.TXT,
				ThumbType: common.NoFile,
				Size:      0x11,
			},
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			testUpload(t, c)
		})
	}
}

func testUpload(t *testing.T, c uploadCase) {
	thread, user := test_db.InsertSampleThread(t)

	body := new(bytes.Buffer)
	w := multipart.NewWriter(body)
	fw, err := w.CreateFormFile("image", c.fileName)
	if err != nil {
		t.Fatal(err)
	}
	f := test.OpenSample(t, c.fileName)
	defer f.Close()
	_, err = io.Copy(fw, f)
	if err != nil {
		t.Fatal(err)
	}
	err = w.Close()
	if err != nil {
		t.Fatal(err)
	}

	sha1Hash, md5Hash := hashImage(t, f)

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Authorization", "Bearer "+user.String())
	req.Header.Set("Content-Length", strconv.Itoa(body.Len()))
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	NewImageUpload(rec, req)
	if c.err != "" {
		test.AssertEquals(t, rec.Body.String(), c.err)
		test.AssertEquals(t, rec.Code, c.code)
		return
	} else if rec.Code != 200 {
		t.Fatalf("failed thumbnailing: %s", rec.Body.String())
		test.AssertEquals(t, rec.Code, c.code)
	}

	c.img.SHA1 = sha1Hash
	c.img.MD5 = md5Hash
	assertImage(t, thread, common.Image{
		ImageCommon: c.img,
		Name:        c.downloadName,
	})
}

func hashImage(t *testing.T, rs io.ReadSeeker) (
	sha1_ common.SHA1Hash,
	md5_ common.MD5Hash,
) {
	t.Helper()

	_, err := hashFile(sha1_[:], rs, sha1.New())
	if err != nil {
		t.Fatal(err)
	}
	_, err = hashFile(md5_[:], rs, md5.New())
	if err != nil {
		t.Fatal(err)
	}
	return
}

func assertImage(t *testing.T, postID uint64, std common.Image) {
	t.Helper()

	var img common.ImageCommon
	err := db.InTransaction(context.Background(), func(tx pgx.Tx) (err error) {
		img, err = db.GetImage(context.Background(), tx, std.SHA1)
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, img, std.ImageCommon)

	var post struct {
		Image *common.Image
	}
	buf, err := db.GetPost(context.Background(), postID)
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal(buf, &post)
	if err != nil {
		t.Fatal(err)
	}
	test.AssertEquals(t, post.Image, &std)
}

func TestUploadHash(t *testing.T) {
	t.Parallel()

	c := uploadCase{
		name:         "PNG",
		fileName:     "sample.png",
		downloadName: "sample",
		code:         200,
		img: common.ImageCommon{
			FileType:    common.PNG,
			ThumbType:   common.WEBP,
			Width:       0x0500,
			Height:      0x02d0,
			ThumbWidth:  0x96,
			ThumbHeight: 0x54,
			Size:        0x09af2e,
		},
	}

	t.Run("initial upload", func(t *testing.T) {
		testUpload(t, c)
	})

	t.Run("hash upload", func(t *testing.T) {
		thread, user := test_db.InsertSampleThread(t)

		f := test.OpenSample(t, c.fileName)
		defer f.Close()
		sha1Hash, md5Hash := hashImage(t, f)

		body := url.Values{
			"id":   {sha1Hash.String()},
			"name": {c.fileName},
		}.
			Encode()
		req := httptest.NewRequest("POST", "/", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+user.String())
		req.Header.Set("Content-Length", strconv.Itoa(len(body)))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rec := httptest.NewRecorder()
		UploadImageHash(rec, req)
		if rec.Code != 200 {
			t.Fatalf("failed hash upload: %s", rec.Body.String())
		}

		c.img.SHA1 = sha1Hash
		c.img.MD5 = md5Hash
		assertImage(t, thread, common.Image{
			ImageCommon: c.img,
			Name:        c.downloadName,
		})
	})
}
