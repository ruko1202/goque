package testutils

import (
	"testing"

	"github.com/ruko1202/goque/internal/storages"
)

// RunMultiDBTests runs the provided test function against each database storage in the taskStorages slice.
// Each test is executed as a subtest named after the database driver.
func RunMultiDBTests(
	t *testing.T,
	taskStorages []storages.AdvancedTaskStorage,
	test func(t *testing.T, storage storages.AdvancedTaskStorage),
) {
	t.Helper()

	for _, storage := range taskStorages {
		t.Run(storage.GetDB().DriverName(), func(t *testing.T) {
			test(t, storage)
		})
	}
}

// RunMultiDBBenchmarks runs the provided benchmark function against each database storage in the taskStorages slice.
// Each benchmark is executed as a sub-benchmark named after the database driver.
func RunMultiDBBenchmarks(
	b *testing.B,
	taskStorages []storages.AdvancedTaskStorage,
	benchmark func(b *testing.B, storage storages.AdvancedTaskStorage),
) {
	b.Helper()

	for _, storage := range taskStorages {
		b.Run(storage.GetDB().DriverName(), func(b *testing.B) {
			benchmark(b, storage)
		})
	}
}
