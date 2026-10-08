package update

import (
	"context"
	"os/exec"
	"path/filepath"
)

func validatePlatformPackage(ctx context.Context, root string) error {
	if exec.CommandContext(ctx, "/usr/bin/codesign", "--verify", "--deep", "--strict", filepath.Join(root, "Clipare.app")).Run() != nil {
		return ErrVerify
	}
	return nil
}
