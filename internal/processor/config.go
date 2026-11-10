package processor

type Config struct {
	// Concurrency is the number of process_torrent queue jobs (each a batch of up to 100 torrents)
	// that are classified and persisted concurrently.
	Concurrency uint
}

func NewDefaultConfig() Config {
	return Config{
		Concurrency: 3,
	}
}
