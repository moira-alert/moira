package s3

// Config is the configuration structure for s3 image store.
type Config struct {
	AccessKeyID string `yaml:"access_key_id"`
	AccessKey   string `yaml:"access_key"`
	Region      string `yaml:"region"`
	Bucket      string `yaml:"bucket"`
}
