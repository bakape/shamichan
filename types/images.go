package types

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
	APNG     bool      `json:"apng,omitempty" gorethink:"apng,omitempty"`
	Audio    bool      `json:"audio,omitempty" gorethink:"audio,omitempty"`
	FileType uint8     `json:"fileType" gorethink:"fileType"`
	Length   uint32    `json:"length,omitempty" gorethink:"length,omitempty"`
	Dims     [4]uint16 `json:"dims" gorethink:"dims"`
	Size     int       `json:"size" gorethink:"size"`
	MD5      string
	SHA1     string
}
