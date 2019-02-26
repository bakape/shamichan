package common

// Supported file formats
const (
	JPEG uint8 = iota
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
)

// Extensions maps internal file types to their canonical file extensions
var Extensions = map[uint8]string{
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
	Audio     bool      `json:"audio"`
	Video     bool      `json:"video"`
	FileType  uint8     `json:"file_type"`
	ThumbType uint8     `json:"thumb_type"`
	Length    uint32    `json:"length"`
	Dims      [4]uint16 `json:"dims"`
	Size      int       `json:"size"`
	Artist    string    `json:"artist"`
	Title     string    `json:"title"`
	MD5       string    `json:"md5"`
	SHA1      string    `json:"sha1"`
}
