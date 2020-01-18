package common

import (
	"encoding/hex"
	"fmt"

	"github.com/jackc/pgtype"
)

var (
	fileTypeStr = [...]string{
		"JPEG",
		"PNG",
		"GIF",
		"WEBM",
		"PDF",
		"SVG",
		"MP4",
		"MP3",
		"OGG",
		"ZIP",
		"7Z",
		"TGZ",
		"TXZ",
		"FLAC",
		"NO_FILE",
		"TXT",
		"WEBP",
		"RAR",
		"CBZ",
		"CBR",
	}
)

// Supported file formats
type FileType uint8

// Supported file formats
const (
	JPEG FileType = iota
	PNG
	GIF
	WEBM
	PDF
	SVG
	MP4
	MP3
	OGG
	ZIP
	SevenZip
	TGZ
	TXZ
	FLAC
	NoFile
	TXT
	WEBP
	RAR
	CBZ
	CBR
)

func (f FileType) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	return append(buf, fileTypeStr[f]...), nil
}

func (f FileType) MarshalText() ([]byte, error) {
	return f.EncodeText(nil, nil)
}

func (f *FileType) DecodeText(_ *pgtype.ConnInfo, src []byte) (err error) {
	return f.UnmarshalText(src)
}

func (f *FileType) UnmarshalText(src []byte) (err error) {
	s := string(src)
	for i, v := range fileTypeStr {
		if s == v {
			*f = FileType(i)
			return
		}
	}
	return fmt.Errorf("invalid FileType: %s", s)
}

// Extensions maps internal file types to their canonical file extensions
var Extensions = map[FileType]string{
	JPEG:     "jpg",
	PNG:      "png",
	GIF:      "gif",
	WEBP:     "webp",
	MP3:      "mp3",
	MP4:      "mp4",
	WEBM:     "webm",
	OGG:      "ogg",
	PDF:      "pdf",
	ZIP:      "zip",
	SevenZip: "7z",
	TGZ:      "tar.gz",
	TXZ:      "tar.xz",
	FLAC:     "flac",
	TXT:      "txt",
	RAR:      "rar",
	CBZ:      "cbz",
	CBR:      "cbr",
}

type errInvalidHashLen int

func (e errInvalidHashLen) Error() string {
	return fmt.Sprintf("invalid hash length: %d", int(e))
}

// MD5 hash capable of being encoded to and decoded from Postgres bytea and JSON
type MD5Hash [16]byte

func (h MD5Hash) EncodeBinary(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	return append(buf, h[:]...), nil
}

func (h *MD5Hash) DecodeBinary(_ *pgtype.ConnInfo, src []byte) (err error) {
	if len(src) != 16 {
		return errInvalidHashLen(len(src))
	}
	copy(h[:], src)
	return
}

func (h MD5Hash) MarshalText() ([]byte, error) {
	dst := make([]byte, 32)
	hex.Encode(dst, h[:])
	return dst, nil
}

func (h *MD5Hash) UnmarshalText(src []byte) (err error) {
	if len(src) != 32 {
		return errInvalidHashLen(len(src) / 2)
	}
	_, err = hex.Decode(h[:], src)
	return
}

func (h MD5Hash) String() string {
	buf, _ := h.MarshalText()
	return string(buf)
}

// SHA1 hash capable of being encoded to and decoded from Postgres bytea and
// JSON
type SHA1Hash [20]byte

func (h SHA1Hash) EncodeBinary(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	return append(buf, h[:]...), nil
}

func (h *SHA1Hash) DecodeBinary(_ *pgtype.ConnInfo, src []byte) (err error) {
	if len(src) != 20 {
		return errInvalidHashLen(len(src))
	}
	copy(h[:], src)
	return
}

func (h SHA1Hash) MarshalText() ([]byte, error) {
	dst := make([]byte, 40)
	hex.Encode(dst, h[:])
	return dst, nil
}

func (h *SHA1Hash) UnmarshalText(src []byte) (err error) {
	if len(src) != 40 {
		return errInvalidHashLen(len(src) / 2)
	}
	_, err = hex.Decode(h[:], src)
	return
}

func (h SHA1Hash) String() string {
	buf, _ := h.MarshalText()
	return string(buf)
}

// Image contains a post's image and thumbnail data
type Image struct {
	Spoiler bool `json:"spoiler"`
	ImageCommon
	Name string `json:"name"`
}

// ImageCommon contains the common data shared between multiple post referencing
// the same image
type ImageCommon struct {
	Audio       bool     `json:"audio"`
	Video       bool     `json:"video"`
	FileType    FileType `json:"file_type" db:"file_type"`
	ThumbType   FileType `json:"thumb_type" db:"thumb_type"`
	Width       uint16   `json:"width" db:",string"`
	Height      uint16   `json:"height" db:",string"`
	ThumbWidth  uint16   `json:"thumb_width" db:"thumb_width,string"`
	ThumbHeight uint16   `json:"thumb_height" db:"thumb_height,string"`
	Duration    uint32   `json:"duration"`
	Size        uint64   `json:"size"`
	Artist      *string  `json:"artist"`
	Title       *string  `json:"title"`
	MD5         MD5Hash  `json:"md5"`
	SHA1        SHA1Hash `json:"sha1"`
}
