package entity

type S3FileEvent struct {
	BucketName     string `json:"bucketName"`
	ObjectKey      string `json:"objectKey"`
	SourceFileName string `json:"sourceFileName"`
}
