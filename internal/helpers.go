package internal

// Throws 'StupidDeveloperException'.
// Panics if given non-nil error.
// Should be used only in case of non-recoverable developer error.
func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
