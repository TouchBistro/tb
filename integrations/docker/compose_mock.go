package docker

type mockCompose struct {
	// TODO(@cszatmary): Implement this for real.
	// This is just a placeholder right now so we don't use real docker-compose in tests.
	Compose
}

// NewMock returns a mock Compose instance that is suitable for tests.
func NewMockCompose() Compose {
	return &mockCompose{}
}
