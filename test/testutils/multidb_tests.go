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
