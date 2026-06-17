package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type lambdaPackage struct {
	name string
	path string
}

func main() {
	distDir := flag.String("dist", "dist", "directory for packaged Lambda artifacts")
	flag.Parse()

	packages := []lambdaPackage{
		{name: "card-onboarding-file-preprocessor", path: "./card-onboarding-file-preprocessor"},
		{name: "card-onboarding-worker", path: "./card-onboarding-worker"},
	}

	if err := os.MkdirAll(*distDir, 0o755); err != nil {
		fatal(err)
	}

	for _, pkg := range packages {
		if err := buildAndZip(*distDir, pkg); err != nil {
			fatal(err)
		}
	}
}

func buildAndZip(distDir string, pkg lambdaPackage) error {
	outputDir := filepath.Join(distDir, pkg.name)
	bootstrapPath := filepath.Join(outputDir, "bootstrap")
	zipPath := filepath.Join(distDir, pkg.name+".zip")

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	cmd := exec.Command("go", "build", "-tags", "lambda.norpc", "-o", bootstrapPath, pkg.path)
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build %s: %w", pkg.name, err)
	}

	if err := os.Chmod(bootstrapPath, 0o755); err != nil {
		return err
	}
	if err := os.RemoveAll(zipPath); err != nil {
		return err
	}

	return zipBootstrap(bootstrapPath, zipPath)
}

func zipBootstrap(bootstrapPath string, zipPath string) error {
	out, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer out.Close()

	archive := zip.NewWriter(out)
	defer archive.Close()

	header := &zip.FileHeader{
		Name:   "bootstrap",
		Method: zip.Deflate,
	}
	header.SetMode(0o755)

	entry, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}

	in, err := os.Open(bootstrapPath)
	if err != nil {
		return err
	}
	defer in.Close()

	_, err = io.Copy(entry, in)
	return err
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
