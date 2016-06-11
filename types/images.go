package types

// Image contains a post's image and thumbnail data
type Image struct {
	Spoiler uint8 `json:"spoiler,omitempty" gorethink:"spoiler,omitempty"`
	ImageCommon
	Imgnm string `json:"imgnm" gorethink:"imgnm"`
}

// ProtoImage stores image data related to the source and thumbnail resources
// themselves. This struct is partially coppied into the image struct on image
// allocattion.
type ProtoImage struct {
	ImageCommon
	Posts int64 `gorethink:"posts,omitempty"`
}

// ImageCommon contains the common fields of both Image and ProtoImage structs
type ImageCommon struct {
	APNG     bool      `json:"apng,omitempty" gorethink:"apng,omitempty"`
	Audio    bool      `json:"audio,omitempty" gorethink:"audio,omitempty"`
	FileType uint8     `json:"fileType" gorethink:"fileType"`
	Length   int32     `json:"length,omitempty" gorethink:"length,omitempty"`
	Dims     [4]uint16 `json:"dims" gorethink:"dims"`
	Size     int64     `json:"size" gorethink:"size"`
	MD5      string
	SHA1     string
}
