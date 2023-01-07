package services

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

func ReadFile(filename string, out interface{}) error {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(file), &out)
}

func FileExist(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func ReplaceFileContent(filename string, content []byte) error {
	return ioutil.WriteFile(filename, content, 0)
}