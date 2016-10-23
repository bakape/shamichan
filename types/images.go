package types

// Supported file formats
const (
	JPEG = iota
	PNG
	GIF
	WEBM
	PDF
	SVG
	MP4
	MP3
	OGG
)

// Extensions maps internal file types to their canonical file extensions
var Extensions = map[uint8]string{
	JPEG: "jpg",
	PNG:  "png",
	GIF:  "gif",
	MP3:  "mp3",
	MP4: "mp4",
	WEBM: "webm",
	OGG:  "ogg",
}

// Image contains a post's image and thumbnail data
type Image struct {
	Spoiler bool `json:"spoiler,omitempty" gorethink:"spoiler,omitempty"`
	ImageCommon
	Name string `json:"name" gorethink:"name"`
}

// ProtoImage stores image data related to the source and thumbnail resources
// themselves. This struct is partially coppied into the image struct on image
// allocattion.
type ProtoImage struct {
	ImageCommon
	Posts int64 `gorethink:"posts"`
}

// ImageCommon contains the common fields of both Image and ProtoImage structs
type ImageCommon struct {
	APNG  bool `json:"apng,omitempty" gorethink:"apng,omitempty"`
	Audio bool `json:"audio,omitempty" gorethink:"audio,omitempty"`

	// Only used for file formats like OGG and MP4 that may or may not contain
	// video
	Video bool `json:"video,omitempty" gorethink:"video,omitempty"`

	FileType uint8     `json:"fileType" gorethink:"fileType"`
	Length   uint32    `json:"length,omitempty" gorethink:"length,omitempty"`
	Dims     [4]uint16 `json:"dims" gorethink:"dims"`
	Size     int       `json:"size" gorethink:"size"`
	MD5      string
	SHA1     string
}
