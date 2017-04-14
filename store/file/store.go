package file

import (
	"bufio"
	"os"
)

type fileStore struct {
	domain string
}

var cNameFile = ".cnames"

func (*fileStore) GetIPAddresses() ([]string, error) {
	panic("implement me")
}

func (*fileStore) SetIPAddresses([]string) error {
	panic("implement me")
}

func (*fileStore) ListCNames() ([]string, error) {
	_, err := os.Stat(cNameFile)
	if err != nil {
		return []string{}, nil
	}
	fp, err := os.Open(cNameFile)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	cNames := []string{}
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		cNames = append(cNames, scanner.Text())
	}
	return cNames, err
}

func (*fileStore) AddCName(domain string) error {
	fp, err := os.OpenFile(cNameFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer fp.Close()
	writer := bufio.NewWriter(fp)
	writer.WriteString(domain + "\n")
	writer.Flush()
	return nil
}

func (fs *fileStore) RemoveCName(domain string) error {
	cNames, err := fs.ListCNames()
	if err != nil {
		return err
	}
	fp, err := os.OpenFile(cNameFile, os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fp.Close()
	writer := bufio.NewWriter(fp)
	for _, cName := range cNames {
		if cName != domain {
			writer.WriteString(cName + "\n")
		}
	}
	writer.Flush()
	return nil
}

func New() *fileStore {
	return &fileStore{}
}
