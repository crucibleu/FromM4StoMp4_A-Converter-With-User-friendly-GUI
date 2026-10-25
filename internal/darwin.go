//go:build darwin

package internal

func GetMP4Box() (string, error) {
	return getCliPath("MP4Box")
}
