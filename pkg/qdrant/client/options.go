package client

// Config holds the client configuration
type Config struct {
	Endpoint string
}

// Option is a functional option for configuring the client
type Option func(*Config)

// WithEndpoint sets a custom API endpoint
func WithEndpoint(endpoint string) Option {
	return func(c *Config) {
		c.Endpoint = endpoint
	}
}
