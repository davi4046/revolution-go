package randutil

func GetRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	var chars []byte

	for i := 0; i < length; i++ {
		char := charset[GetRandomInt(0, len(charset)-1)]
		chars = append(chars, char)
	}

	return string(chars)
}
