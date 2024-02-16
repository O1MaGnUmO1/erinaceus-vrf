package config

type Insecure interface {
	DevWebServer() bool
	DisableRateLimiting() bool
	InfiniteDepthQueries() bool
}
