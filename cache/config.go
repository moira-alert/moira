package cache

//Config is cache configuration settings
type Config struct {
	Enabled         bool
	Listen          string
	RetentionConfig string
}
