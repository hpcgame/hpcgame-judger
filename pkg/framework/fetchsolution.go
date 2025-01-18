package framework

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mholt/archiver/v3"
)

func UnarchiveFile(archivePath string, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	var u archiver.Unarchiver
	if b, err := archiver.DefaultTarGz.Match(f); b && (err != nil) {
		u = archiver.NewTarGz()
	} else if b, err := archiver.DefaultTarBz2.Match(f); b && (err != nil) {
		u = archiver.NewTarBz2()
	} else if b, err := archiver.DefaultTarXz.Match(f); b && (err != nil) {
		u = archiver.NewTarXz()
	} else {
		u, err = archiver.ByHeader(f)
		if err != nil {
			return err
		}
	}
	return u.Unarchive(archivePath, destPath)
}

func FetchSolution(destPath string) {
	solutionURL := os.Getenv("SOLUTION_URL")

	if solutionURL == "" {
		PanicString("SOLUTION_URL is not set")
	}

	resp := Must(http.Get(solutionURL))
	defer resp.Body.Close()

	tmpPath := filepath.Join(destPath, "temp_solution.zip")

	f := Must(os.Create(tmpPath))
	Must(io.Copy(f, resp.Body))
	NilOrPanic(f.Close())

	NilOrPanic(UnarchiveFile(tmpPath, destPath))
	NilOrPanic(os.Remove(tmpPath))
}
