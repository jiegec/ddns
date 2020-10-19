package main

// DDNSProvider is an interface for ddns providers
type DDNSProvider interface {
	Set(name string, value string, record string) error
	Get(name string, record string) ([]string, error)
}
