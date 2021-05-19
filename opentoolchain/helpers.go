package opentoolchain

func getStringPtr(s string) *string {
	val := s
	return &val
}
