package task

import (
	"os"
	"path/filepath"
	"time"

	"github.com/zdnscloud/cement/log"

	"github.com/linkingthing/ddi-controller/pkg/util"
)

func init() {
	go func() {
		ticker := time.NewTicker(time.Hour * 24)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := removerExpireCsv("*.csv"); err != nil {
					log.Errorf("removerExpireCsv error:%s\n", err.Error())
				}
			}
		}
	}()
}

func removerExpireCsv(patten string) error {
	nowTime := time.Now()
	return filepath.Walk(util.FileRootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if match, _ := filepath.Match(patten, info.Name()); match && nowTime.After(info.ModTime().AddDate(0, 0, 7)) {
			return os.Remove(path)
		}

		return nil
	})
}
