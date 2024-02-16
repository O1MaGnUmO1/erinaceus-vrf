package config

type Feature interface {
	UICSAKeys() bool
	LogPoller() bool
}
