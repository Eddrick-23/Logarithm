package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

func GenerateFileName(path string, filename string) (string, error) {
	n, err := NumFilesInFolder(path)
	if err != nil {
		return "", err
	}

	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	ext := filepath.Ext(filename)

	for suffix := 0; suffix <= n; suffix++ {
		name := base
		if suffix != 0 {
			name += strconv.Itoa(suffix)
		}
		name += ext

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

func CreateResultFile(outDir string, filename string) (*os.File, error) {
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

	filename, err = GenerateFileName(outDir, filename)
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

// Write a signal file so bash script knows when load generation is completed
func CreateReadyFile(outDir, filename string) error {
	return os.WriteFile(filepath.Join(outDir, filename), []byte("ready"), 0644)
}

// Delete file if it exists. If file does not exist, returns nil error
func DeleteFile(outDir, filename string) error {
	path := filepath.Join(outDir, filename)
	ok, err := FileExists(path)

	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	return os.Remove(path)
}
