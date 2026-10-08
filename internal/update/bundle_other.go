//go:build !darwin

package update

import "context"

func validatePlatformPackage(context.Context, string) error { return nil }
