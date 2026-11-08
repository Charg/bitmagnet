package metainforequester

import "time"

type Config struct {
	DialTimeout    time.Duration
	RequestTimeout time.Duration
	KeyMutexSize   int
}

func NewDefaultConfig() Config {
	return Config{
		DialTimeout:    2 * time.Second,
		RequestTimeout: 6 * time.Second,
		KeyMutexSize:   1000,
	}
}
