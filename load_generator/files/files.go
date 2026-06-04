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

func CreateResultFile() (*os.File, error) {
	const resultsDir = "results"
	dir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v", err)
		return nil, err
	}
	exists, err := FolderExists(filepath.Join(dir, resultsDir))
	if err != nil {
		fmt.Printf("Could not verify if results folder exists: %v", err)
		return nil, err
	}

	if !exists {
		// 0755 give read/write/execute to owner
		if err := os.Mkdir(filepath.Join(dir, resultsDir), 0755); err != nil {
			fmt.Printf("failed to create results directory: %v", err)
			return nil, err
		}
	}

	count, err := NumFilesInFolder(filepath.Join(dir, resultsDir))
	if err != nil {
		return nil, fmt.Errorf("could not count result files: %w", err)
	}

	filename := "results"
	if count > 0 {
		filename = filename + strconv.Itoa(count)
	}
	filename += ".bin"

	file, err := os.Create(filepath.Join(dir, "results", filename))
	if err != nil {
		fmt.Printf("error creating results file: %v", err)
		return nil, err
	}
	return file, nil
}
