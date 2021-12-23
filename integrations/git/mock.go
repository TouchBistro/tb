package git

type mockGit struct {
	// TODO(@cszatmary): Implement this for real.
	// This is just a placeholder right now so we don't use real git in tests.
	Git
}

// NewMock returns a mock Git instance that is suitable for tests.
func NewMock() Git {
	return &mockGit{}
}
