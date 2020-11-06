package util

import (
	"bytes"
	"strconv"
	"time"

	"github.com/zdnscloud/cement/slice"
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"
)

const (
	FilterTimeFrom = "from"
	FilterTimeTo   = "to"
	TimeFromSuffix = " 00:00"
	TimeToSuffix   = " 23:59"
)

func GenSqlAndArgsByFileters(table restdb.ResourceType, filterNames []string, from int, filters []restresource.Filter) (string, []interface{}) {
	now := time.Now()
	timeFrom := now.AddDate(0, 0, -from).Format("2006-01-02")
	timeTo := now.Format("2006-01-02")
	conditions := make(map[string]string)
	for _, filter := range filters {
		if slice.SliceIndex(filterNames, filter.Name) != -1 {
			if value, ok := GetFilterValueWithEqModifierFromFilter(filter); ok {
				conditions[filter.Name+" = $"] = value
			}
		} else {
			switch filter.Name {
			case FilterTimeFrom:
				if from, ok := GetFilterValueWithEqModifierFromFilter(filter); ok {
					timeFrom = from
				}
			case FilterTimeTo:
				if to, ok := GetFilterValueWithEqModifierFromFilter(filter); ok {
					timeTo = to
				}
			}
		}
	}

	var buf bytes.Buffer
	buf.WriteString("select * from gr_")
	buf.WriteString(string(table))
	buf.WriteString(" where ")
	var args []interface{}
	for cond, arg := range conditions {
		args = append(args, arg)
		buf.WriteString(cond)
		buf.WriteString(strconv.Itoa(len(args)))
		buf.WriteString(" and ")
	}

	buf.WriteString("create_time between ")
	buf.WriteString("'")
	buf.WriteString(timeFrom)
	buf.WriteString(TimeFromSuffix)
	buf.WriteString("'")
	buf.WriteString(" and ")
	buf.WriteString("'")
	buf.WriteString(timeTo)
	buf.WriteString(TimeToSuffix)
	buf.WriteString("'")
	buf.WriteString(" order by create_time desc")
	return buf.String(), args
}
