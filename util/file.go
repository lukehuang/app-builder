package util

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

func ReadDirContent(dirPath string) ([]string, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	files, err := dir.Readdirnames(0)
	return files, CloseAndCheckError(err, dir)
}

func ReadFile(file string, size int) ([]byte, error) {
	reader, err := os.Open(file)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	result := make([]byte, size)
	_, err = reader.Read(result)
	return result, CloseAndCheckError(err, reader)
}

func CloseAndCheckError(err error, closable io.Closer) error {
	closeErr := closable.Close()
	if err != nil {
		return errors.WithStack(err)
	}
	if closeErr != nil {
		return errors.WithStack(closeErr)
	}
	return nil
}

func RemoveByGlob(fileGlob string) error {
	if !strings.HasSuffix(fileGlob, "*") {
		return errors.WithStack(os.RemoveAll(fileGlob))
	}

	dir := filepath.Dir(fileGlob)
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, file := range files {
		matched, err := filepath.Match(fileGlob, file.Name())
		if err != nil {
			return err
		}
		if !matched {
			continue
		}

		absoluteChildFile := filepath.Join(dir, file.Name())
		if file.IsDir() {
			err = os.RemoveAll(absoluteChildFile)
		} else {
			err = syscall.Unlink(absoluteChildFile)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// go doesn't provide native copy operation (CoW)
func copyDir(from string, to string) error {
	fromInfo, err := os.Stat(from)
	if err != nil {
		return errors.WithStack(err)
	}

	if !fromInfo.IsDir() {
		return errors.Errorf("Source directory \"%s\" must be a directory", from)
	}

	err = os.MkdirAll(to, fromInfo.Mode())
	if err != nil {
		return errors.WithStack(err)
	}

	fileNames, err := ReadDirContent(from)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, name := range fileNames {
		if name == "default" {
			continue
		}

		err = CopyDirOrFile(filepath.Join(from, name), filepath.Join(to, name))
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func CopyDirOrFile(from string, to string) error {
	fromInfo, err := os.Stat(from)
	if err != nil {
		return errors.WithStack(err)
	}

	if fromInfo.IsDir() {
		return copyDir(from, to)
	} else {
		return copyFile(from, to, fromInfo)
	}
}

func copyFile(from string, to string, fromInfo os.FileInfo) error {
	s, err := os.Open(from)
	if err != nil {
		return errors.WithStack(err)
	}

	defer s.Close()
	d, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fromInfo.Mode())
	if err != nil {
		s.Close()
		return errors.WithStack(err)
	}

	_, err = io.Copy(d, s)
	if err != nil {
		d.Close()
		return errors.WithStack(err)
	}

	err = d.Close()
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}