//go:build !zscalerctl_testhooks

package dump

import "os"

func registerRootPathForTestHooks(*os.Root, string) {}

func rootPathForTestHooks(root *os.Root) string { return root.Name() }

func runInventoryDirectoryTestHook(string, string) {}

func runInventoryAfterReadDirTestHook(string, string) {}
