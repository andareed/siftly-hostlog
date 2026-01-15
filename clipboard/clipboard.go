package clipboard

func Copy(text string) error {
	return copyOSC52(text)
}
