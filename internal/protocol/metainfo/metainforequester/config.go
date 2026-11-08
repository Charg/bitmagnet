package metainforequester

import "time"

type Config struct {
	DialTimeout    time.Duration
	RequestTimeout time.Duration
	KeyMutexSize   int
	// MaxConcurrentConnections caps the number of concurrent outgoing TCP connections used to fetch meta info.
	// Many concurrent connections can overwhelm the connection tracking of consumer routers. 0 disables the cap.
	MaxConcurrentConnections uint
}

func NewDefaultConfig() Config {
	return Config{
		DialTimeout:    2 * time.Second,
		RequestTimeout: 6 * time.Second,
		KeyMutexSize:   1000,
		// matches the peak before meta info was requested from multiple peers concurrently:
		MaxConcurrentConnections: 100,
	}
}
