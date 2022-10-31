package updater

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// TarGZExtract extracts a archive from the file inputFilePath.
// It tries to create the directory structure outputFilePath contains if it doesn't exist.
// It returns potential errors to be checked or nil if everything works.
func TarGZExtract(inputFilePath, outputFilePath string) (err error) {
	outputFilePath = stripTrailingSlashes(outputFilePath)
	inputFilePath, outputFilePath, err = makeAbsolute(inputFilePath, outputFilePath)
	if err != nil {
		return err
	}
	undoDir, err := mkdirAll(outputFilePath, 0750)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			undoDir()
		}
	}()

	return extract(inputFilePath, outputFilePath)
}

// Creates all directories with os.MakedirAll and returns a function to remove the first created directory so cleanup is possible.
func mkdirAll(dirPath string, perm os.FileMode) (func(), error) {
	var undoDir string

	for p := dirPath; ; p = filepath.Dir(p) {
		finfo, err := os.Stat(p)
		if err == nil {
			if finfo.IsDir() {
				break
			}

			finfo, err = os.Lstat(p)
			if err != nil {
				return nil, err
			}

			if finfo.IsDir() {
				break
			}

			return nil, fmt.Errorf("mkdirAll (%s): %v", p, syscall.ENOTDIR)
		}

		if os.IsNotExist(err) {
			undoDir = p
		} else {
			return nil, err
		}
	}

	if undoDir == "" {
		return func() {}, nil
	}

	if err := os.MkdirAll(dirPath, perm); err != nil {
		return nil, err
	}

	return func() {
		if err := os.RemoveAll(undoDir); err != nil {
			panic(err)
		}
	}, nil
}

// Remove trailing slash if any.
func stripTrailingSlashes(path string) string {
	if len(path) > 0 && path[len(path)-1] == '/' {
		path = path[0 : len(path)-1]
	}

	return path
}

// Make input and output paths absolute.
func makeAbsolute(inputFilePath, outputFilePath string) (string, string, error) {
	inputFilePath, err := filepath.Abs(inputFilePath)
	if err == nil {
		outputFilePath, err = filepath.Abs(outputFilePath)
	}

	return inputFilePath, outputFilePath, err
}

// Write path without the prefix in subPath to tar writer.
func writeTarGz(path string, tarWriter *tar.Writer, fileInfo os.FileInfo, subPath string) error {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return err
	}

	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	evaledPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}

	subPath, err = filepath.EvalSymlinks(subPath)
	if err != nil {
		return err
	}

	link := ""
	if evaledPath != path {
		link = evaledPath
	}

	header, err := tar.FileInfoHeader(fileInfo, link)
	if err != nil {
		return err
	}
	header.Name = evaledPath[len(subPath):]

	err = tarWriter.WriteHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(tarWriter, file)
	if err != nil {
		return err
	}

	return err
}

// Extract the file in filePath to directory.
func extract(filePath string, directory string) error {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return err
	}

	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	gzipReader, err := gzip.NewReader(bufio.NewReader(file))
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	// Post extraction directory permissions & timestamps
	type DirInfo struct {
		Path   string
		Header *tar.Header
	}

	// slice to add all extracted directory info for post-processing
	postExtraction := []DirInfo{}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fileInfo := header.FileInfo()
		// paths could contain a '..', is used in a file system operations
		if strings.Contains(fileInfo.Name(), "..") {
			continue
		}
		dir := filepath.Join(directory, filepath.Dir(header.Name))
		filename := filepath.Join(dir, fileInfo.Name())

		if fileInfo.IsDir() {
			// create the directory 755 in case writing permissions prohibit writing before files added
			if err := os.MkdirAll(filename, 0750); err != nil {
				return err
			}

			// set file ownership (if allowed)
			// Chtimes() && Chmod() only set after once extraction is complete
			os.Chown(filename, header.Uid, header.Gid) // #nosec

			// add directory info to slice to process afterwards
			postExtraction = append(postExtraction, DirInfo{filename, header})
			continue
		}

		// make sure parent directory exists (may not be included in tar)
		if !fileInfo.IsDir() && !isDir(dir) {
			err = os.MkdirAll(dir, 0750)
			if err != nil {
				return err
			}
		}

		file, err := os.Create(filepath.Clean(filename))
		if err != nil {
			return err
		}

		writer := bufio.NewWriter(file)

		buffer := make([]byte, 4096)
		for {
			n, err := tarReader.Read(buffer)
			if err != nil && err != io.EOF {
				panic(err)
			}
			if n == 0 {
				break
			}

			_, err = writer.Write(buffer[:n])
			if err != nil {
				return err
			}
		}

		err = writer.Flush()
		if err != nil {
			return err
		}

		err = file.Close()
		if err != nil {
			return err
		}

		// set file permissions, timestamps & uid/gid
		os.Chmod(filename, os.FileMode(header.Mode))            // #nosec
		os.Chtimes(filename, header.AccessTime, header.ModTime) // #nosec
		os.Chown(filename, header.Uid, header.Gid)              // #nosec
	}

	if len(postExtraction) > 0 {
		for _, dir := range postExtraction {
			os.Chtimes(dir.Path, dir.Header.AccessTime, dir.Header.ModTime) // #nosec
			os.Chmod(dir.Path, dir.Header.FileInfo().Mode().Perm())         // #nosec
		}
	}

	return nil
}
