package cache

// SetNowUnixForTest overrides cache's clock and returns a restore function.
func SetNowUnixForTest(f func() int64) func() {
	prev := nowUnix
	nowUnix = f
	return func() {
		nowUnix = prev
	}
}
