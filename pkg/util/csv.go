package util

import (
	"encoding/csv"
	"fmt"
	"os"
)

const (
	UTF8BOM      = "\xEF\xBB\xBF"
	TimeFormat   = "2006-01-02 15:04:05"
	FileRootPath = "/opt/website/"
	CSVFilePath  = FileRootPath + "%s.csv"
)

func GenCSVFile(filepath string, tableHeader []string, contents [][]string) error {
	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create csv file %s failed: %s", filepath, err.Error())
	}

	defer file.Close()
	file.WriteString(UTF8BOM)
	w := csv.NewWriter(file)
	if err := w.Write(tableHeader); err != nil {
		return fmt.Errorf("write table header to csv file %s failed: %s", filepath, err.Error())
	}

	if err := w.WriteAll(contents); err != nil {
		return fmt.Errorf("write data to csv file %s failed: %s", filepath, err.Error())
	}

	w.Flush()
	return nil
}
