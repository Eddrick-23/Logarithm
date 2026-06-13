package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func FileExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil

}

func GenerateFileName(path string, prefix string) (string, error) {
	n, err := NumFilesInFolder(path)
	if err != nil {
		return "", err
	}

	for suffix := 0; suffix <= n; suffix++ {
		name := prefix
		if suffix != 0 {
			name += strconv.Itoa(suffix)
		}
		name += ".bin"

		exists, err := FileExists(filepath.Join(path, name))
		if err != nil {
			return "", err
		}

		if !exists {
			return name, nil
		}
	}

	return "", fmt.Errorf("no valid name found")
}

func FolderExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

func NumFilesInFolder(path string) (int, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return 0, err
	}

	return len(files), nil
}

func CreateResultFile(outDir string) (*os.File, error) {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v", err)
		return nil, err
	}
	exists, err := FolderExists(filepath.Join(dir, outDir))
	if err != nil {
		fmt.Printf("Could not verify if target folder exists: %v", err)
		return nil, err
	}

	if !exists {
		// 0755 give read/write/execute to owner
		if err := os.MkdirAll(filepath.Join(dir, outDir), 0755); err != nil {
			fmt.Printf("failed to create results directory: %v", err)
			return nil, err
		}
	}

	filename, err := GenerateFileName(outDir, "results")
	if err != nil {
		fmt.Printf("could not find suitable result file name: %v", err)
		return nil, err
	}

	file, err := os.Create(filepath.Join(dir, outDir, filename))
	if err != nil {
		fmt.Printf("error creating results file: %v", err)
		return nil, err
	}
	return file, nil
}
