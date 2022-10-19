package conversion

func MBtoGB(f float64) float64 {
	return f / 1024
}

func TBtoGB(f float64) float64 {
	return f * 1024
}
