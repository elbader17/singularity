package updater

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Builder builds the application locally
type Builder struct {
	repoPath   string
	mainPath   string
	outputPath string
}

func NewBuilder(repoPath, mainPath, outputPath string) *Builder {
	return &Builder{
		repoPath:   repoPath,
		mainPath:   mainPath,
		outputPath: outputPath,
	}
}

// Build compiles the application
func (b *Builder) Build(ctx context.Context) error {
	// Ensure output directory exists
	outputDir := filepath.Dir(b.outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build command
	cmd := exec.CommandContext(ctx, "go", "build", "-o", b.outputPath, b.mainPath)
	cmd.Dir = b.repoPath
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build failed: %w - %s", err, string(output))
	}

	// Ensure the output is executable
	if err := os.Chmod(b.outputPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	return nil
}

// BuildWithVersion builds with version info injected
func (b *Builder) BuildWithVersion(ctx context.Context, version, commit string) error {
	// Ensure output directory exists
	outputDir := filepath.Dir(b.outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build command with ldflags
	cmd := exec.CommandContext(ctx, "go", "build",
		"-ldflags", fmt.Sprintf("-X singularity/internal/version.Version=%s -X singularity/internal/version.Commit=%s", version, commit),
		"-o", b.outputPath,
		b.mainPath)
	cmd.Dir = b.repoPath
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build failed: %w - %s", err, string(output))
	}

	// Ensure the output is executable
	if err := os.Chmod(b.outputPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	return nil
}

// GetOutputPath returns the path where the binary will be built
func (b *Builder) GetOutputPath() string {
	return b.outputPath
}
