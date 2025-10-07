package library

func Opt[K any](v K) *K {
	return &v
}
