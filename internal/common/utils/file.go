package utils

import (
	"os"
	"path"
)

func CreateFileIfNotExist(filePath string) error {
    _, err := os.Stat(filePath)
    if os.IsNotExist(err) {
        err = os.MkdirAll(path.Dir(filePath), os.ModePerm)
        if err != nil {
            return err
        }
        f, err := os.Create(filePath)
        if err != nil {
            return err
        }
        defer f.Close()
    }
    return err
}

