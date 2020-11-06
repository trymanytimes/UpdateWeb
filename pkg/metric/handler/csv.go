package handler

import (
	"fmt"
	"time"

	"github.com/linkingthing/ddi-controller/pkg/util"
)

func exportFile(ctx *MetricContext, contents [][]string) (string, error) {
	filepath := fmt.Sprintf(util.CSVFilePath, ctx.NodeIP+"-"+ctx.MetricName+"-"+time.Now().Format(util.TimeFormat))
	return filepath, util.GenCSVFile(filepath, ctx.TableHeader, contents)
}
