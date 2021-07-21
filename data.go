package pgslap

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/winebarrel/randstr"
)

type AutoGenerateSqlLoadType string

const (
	LoadTypeMixed         = AutoGenerateSqlLoadType("mixed")  // require pre-populated data
	LoadTypeUpdate        = AutoGenerateSqlLoadType("update") // require pre-populated data
	LoadTypeWrite         = AutoGenerateSqlLoadType("write")
	LoadTypeKey           = AutoGenerateSqlLoadType("key")  // require pre-populated data
	LoadTypeRead          = AutoGenerateSqlLoadType("read") // require pre-populated data
	AutoGenerateTableName = "t1"
)

type DataOpts struct {
	LoadType               AutoGenerateSqlLoadType
	GuidPrimary            bool
	NumberSecondaryIndexes int
	CommitRate             int
	MixedSelRatio          int
	MixedInsRatio          int
	NumberIntCols          int
	IntColsIndex           bool
	NumberCharCols         int
	CharColsIndex          bool
	Queries                []string `json:"-"`
	PreQueries             []string
}

type Data struct {
	*DataOpts
	randSrc   rand.Source
	idList    []string
	idIdx     int
	mixedIdx  int
	commitCnt int
	committed bool
	queryIdx  int
}

func newData(opts *DataOpts, idList []string) (data *Data) {
	data = &Data{
		DataOpts: opts,
		randSrc:  rand.NewSource(time.Now().UnixNano()),
		idList:   idList,
	}

	return
}

func (data *Data) initStmts() []string {
	stmts := []string{}

	if len(data.PreQueries) > 0 {
		stmts = append(stmts, data.PreQueries...)
	}

	if data.CommitRate > 0 {
		stmts = append(stmts, "BEGIN")
	}

	return stmts
}

func (data *Data) next() (string, []interface{}) {
	if data.CommitRate > 0 {
		if data.commitCnt == data.CommitRate {
			data.commitCnt = 0
			data.committed = true
			return "COMMIT", []interface{}{}
		}

		if data.committed {
			data.committed = false
			return "BEGIN", []interface{}{}
		}

		data.commitCnt++
	}

	if len(data.Queries) > 0 {
		q := data.Queries[data.queryIdx]
		data.queryIdx++

		if data.queryIdx == len(data.Queries) {
			data.queryIdx = 0
		}

		return q, []interface{}{}
	}

	switch data.LoadType {
	case LoadTypeMixed:
		var stmt string
		var args []interface{}
		if data.mixedIdx < data.MixedSelRatio {
			stmt, args = data.buildSelectStmt(true)
		} else {
			stmt, args = data.buildInsertStmt()
		}

		data.mixedIdx++

		if data.mixedIdx >= data.MixedSelRatio+data.MixedInsRatio {
			data.mixedIdx = 0
		}

		return stmt, args
	case LoadTypeUpdate:
		return data.buildUpdateStmt()
	case LoadTypeWrite:
		return data.buildInsertStmt()
	case LoadTypeKey:
		return data.buildSelectStmt(true)
	case LoadTypeRead:
		return data.buildSelectStmt(false)
	default:
		panic("Failed to generate SQL statement: invalid load type: " + data.LoadType)
	}
}

func (data *Data) buildCreateTableStmt() (string, []string) {
	indices := []string{}
	sb := strings.Builder{}
	sb.WriteString("CREATE TABLE " + AutoGenerateTableName + " (id ")

	if data.GuidPrimary {
		sb.WriteString("uuid PRIMARY KEY DEFAULT gen_random_uuid()")
	} else {
		sb.WriteString("serial PRIMARY KEY")
	}

	for i := 1; i <= data.NumberSecondaryIndexes; i++ {
		fmt.Fprintf(&sb, ",id%d uuid UNIQUE", i)
	}

	for i := 1; i <= data.NumberIntCols; i++ {
		fmt.Fprintf(&sb, ",intcol%d int", i)

		if data.IntColsIndex {
			indices = append(indices, fmt.Sprintf("CREATE INDEX ON "+AutoGenerateTableName+"(intcol%d)", i))
		}
	}

	for i := 1; i <= data.NumberCharCols; i++ {
		fmt.Fprintf(&sb, ",charcol%d varchar(128)", i)

		if data.CharColsIndex {
			indices = append(indices, fmt.Sprintf("CREATE INDEX ON "+AutoGenerateTableName+"(charcol%d)", i))
		}
	}

	sb.WriteString(")")

	return sb.String(), indices
}

func (data *Data) buildSelectStmt(key bool) (string, []interface{}) {
	args := []interface{}{}
	sb := strings.Builder{}
	sb.WriteString("SELECT ")

	for i := 1; i <= data.NumberIntCols; i++ {
		if i >= 2 {
			sb.WriteString(",")
		}

		fmt.Fprintf(&sb, "intcol%d", i)
	}

	for i := 1; i <= data.NumberCharCols; i++ {
		if data.NumberIntCols >= 1 || i >= 2 {
			sb.WriteString(",")
		}

		fmt.Fprintf(&sb, "charcol%d", i)
	}

	sb.WriteString(" FROM " + AutoGenerateTableName)

	if key {
		fmt.Fprintf(&sb, " WHERE id = $1")
		args = append(args, data.nextId())
	}

	return sb.String(), args
}

func (data *Data) buildInsertStmt() (string, []interface{}) {
	args := []interface{}{}
	phIdx := 1
	sb := strings.Builder{}
	sb.WriteString("INSERT INTO " + AutoGenerateTableName + " VALUES (DEFAULT")

	for i := 1; i <= data.NumberSecondaryIndexes; i++ {
		sb.WriteString(",gen_random_uuid()")
	}

	for i := 1; i <= data.NumberIntCols; i++ {
		fmt.Fprintf(&sb, ",$%d", phIdx)
		phIdx++
		num := data.randSrc.Int63() >> 32
		args = append(args, num)
	}

	for i := 1; i <= data.NumberCharCols; i++ {
		fmt.Fprintf(&sb, ",$%d", phIdx)
		phIdx++
		args = append(args, randstr.String(data.randSrc, 128))
	}

	sb.WriteString(")")

	return sb.String(), args
}

func (data *Data) buildUpdateStmt() (string, []interface{}) {
	args := []interface{}{}
	phIdx := 1
	sb := strings.Builder{}
	sb.WriteString("UPDATE " + AutoGenerateTableName + " SET ")

	for i := 1; i <= data.NumberIntCols; i++ {
		if i >= 2 {
			sb.WriteString(",")
		}

		fmt.Fprintf(&sb, "intcol%d = $%d", i, phIdx)
		phIdx++
		v := data.randSrc.Int63() >> 32
		args = append(args, v)
	}

	for i := 1; i <= data.NumberCharCols; i++ {
		if data.NumberIntCols >= 1 || i >= 2 {
			sb.WriteString(",")
		}

		fmt.Fprintf(&sb, "charcol%d = $%d", i, phIdx)
		phIdx++
		args = append(args, randstr.String(data.randSrc, 128))
	}

	fmt.Fprintf(&sb, " WHERE id = $%d", phIdx)
	args = append(args, data.nextId())

	return sb.String(), args
}

func (data *Data) nextId() string {
	if data.idIdx >= len(data.idList) {
		data.idIdx = 0
	}

	id := data.idList[data.idIdx]
	data.idIdx++

	return id
}
