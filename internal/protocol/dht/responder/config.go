package responder

type Config struct {
	// MaxRequestsPerSecond is the overall limit of incoming DHT requests that will be answered per second.
	// Raising it improves passive discovery of nodes and info hashes, at the cost of more network flows.
	MaxRequestsPerSecond uint `validate:"min=1"`
	// MaxRequestsBurst is the burst size allowed above MaxRequestsPerSecond.
	MaxRequestsBurst uint `validate:"min=1"`
}

func NewDefaultConfig() Config {
	return Config{
		MaxRequestsPerSecond: 100,
		MaxRequestsBurst:     20,
	}
}
