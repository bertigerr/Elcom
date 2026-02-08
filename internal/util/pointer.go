package util

func StringPtr(s string) *string {
	return &s
}

func IntPtr(v int) *int {
	return &v
}

func FloatPtr(v float64) *float64 {
	return &v
}
