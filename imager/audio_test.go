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

const mp3Length uint32 = 1

func TestProcessMP3NoCover(t *testing.T) {
	t.Parallel()

	const name = "sample.mp3"

	thread, user := test_db.InsertSampleThread(t)

	body := new(bytes.Buffer)
	w := multipart.NewWriter(body)
	fw, err := w.CreateFormFile("image", name)
	if err != nil {
		t.Fatal(err)
	}
	f := test.OpenSample(t, name)
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

	var img common.ImageCommon
	err = db.InTransaction(context.Background(), func(tx pgx.Tx) (err error) {
		img, err = db.GetImage(context.Background(), tx, sha1Hash)
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	std := common.ImageCommon{
		Audio:     true,
		FileType:  common.MP3,
		ThumbType: common.NoFile,
		Duration:  1,
		Size:      0x782c,
		SHA1:      sha1Hash,
		MD5:       md5Hash,
	}
	test.AssertEquals(t, img, std)

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
		Name:        name,
		ImageCommon: std,
	})
}

// func TestProcessMP3(t *testing.T) {
// 	t.Parallel()

// 	var img common.ImageCommon
// 	f := test.OpenSample(t, "with_cover.mp3")
// 	defer f.Close()
// 	thumb, err := processFile(f, &img, dummyOpts)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	assertThumbnail(t, thumb)
// 	assertDims(t, img.Dims, assets.StdDims["png"])
// 	assertLength(t, img.Duration, mp3Length)
// }
