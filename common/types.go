package common

// Struct that represents environment variable.
// Will be parsed by executor in form of "Key"="Value".
type EnvironmentEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
