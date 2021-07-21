package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"pgslap"
	"strconv"
	"strings"
	"time"

	"github.com/integrii/flaggy"
	"github.com/jackc/pgx/v4"
)

var version string

const (
	DefaultTime                   = 60
	DefaultDBName                 = "pgslap"
	DefaultNumberPrePopulatedData = 100
	DefaultLoadType               = string(pgslap.LoadTypeMixed)
	DefaultNumberIntCols          = 1
	DefaultNumberCharCols         = 1
	DefaultDelimiter              = ";"
)

type Flags struct {
	pgslap.TaskOpts
	pgslap.DataOpts
	pgslap.RecorderOpts
}

func parseFlags() (flags *Flags) {
	flaggy.SetVersion(version)
	flaggy.SetDescription("PostgreSQL load testing tool like mysqlslap.")
	flags = &Flags{}
	var url string
	flaggy.String(&url, "u", "url", "Database URL, e.g. 'postgres://username:password@localhost:5432'.")
	flags.NAgents = 1
	flaggy.Int(&flags.NAgents, "n", "nagents", "Number of agents.")
	argTime := DefaultTime
	flaggy.Int(&argTime, "t", "time", "Test run time (sec). Zero is infinity.")
	flaggy.Int(&flags.NumberQueriesToExecute, "", "number-queries", "Number of queries to execute per agent. Zero is infinity.")
	flaggy.Int(&flags.Rate, "r", "rate", "Rate limit for each agent (qps). Zero is unlimited.")
	flaggy.Bool(&flags.AutoGenerateSql, "a", "auto-generate-sql", "Automatically generate SQL to execute.")
	flaggy.Bool(&flags.GuidPrimary, "", "auto-generate-sql-guid-primary", "Use GUID as the primary key of the table to be created.")
	var queries string
	flaggy.String(&queries, "q", "query", "SQL to execute. (file or string)")
	flags.NumberPrePopulatedData = DefaultNumberPrePopulatedData
	flaggy.Int(&flags.NumberPrePopulatedData, "", "auto-generate-sql-write-number", "Number of rows to be pre-populated for each agent.")
	strLoadType := DefaultLoadType
	flaggy.String(&strLoadType, "l", "auto-generate-sql-load-type", "Test load type: 'mixed', 'update', 'write', 'key', or 'read'.")
	flaggy.Int(&flags.NumberSecondaryIndexes, "", "auto-generate-sql-secondary-indexes", "Number of secondary indexes in the table to be created.")
	flaggy.Int(&flags.CommitRate, "", "commit-rate", "Commit every X queries.")
	mixedSelInsRatio := "1:1"
	flaggy.String(&mixedSelInsRatio, "", "mixed-sel-ins-ratio", "Mixed load type 'SELECT:INSERT' ratio.")
	flags.NumberCharCols = DefaultNumberCharCols
	flaggy.Int(&flags.NumberCharCols, "x", "number-char-cols", "Number of VARCHAR columns in the table to be created.")
	flaggy.Bool(&flags.CharColsIndex, "", "char-cols-index", "Create indexes on VARCHAR columns in the table to be created.")
	flags.NumberIntCols = DefaultNumberIntCols
	flaggy.Int(&flags.NumberIntCols, "y", "number-int-cols", "Number of INT columns in the table to be created.")
	flaggy.Bool(&flags.IntColsIndex, "", "int-cols-index", "Create indexes on INT columns in the table to be created.")
	var preqs string
	flaggy.String(&preqs, "", "pre-query", "Queries to be pre-executed for each agent.")
	var creates string
	flaggy.String(&creates, "", "create", "SQL for creating custom tables. (file or string)")
	flaggy.Bool(&flags.DropExistingDatabase, "", "drop-db", "Forcibly delete the existing DB.")
	flaggy.Bool(&flags.NoDropDatabase, "", "no-drop", "Do not drop database after testing.")
	hinterval := "0"
	flaggy.String(&hinterval, "", "hinterval", "Histogram interval, e.g. '100ms'.")
	delimiter := DefaultDelimiter
	flaggy.String(&delimiter, "F", "delimiter", "SQL statements delimiter.")
	flaggy.Bool(&flags.OnlyPrint, "", "only-print", "Just print SQL without connecting to DB.")
	flaggy.Bool(&flags.NoProgress, "", "no-progress", "Do not show progress.")
	flaggy.Parse()

	if len(os.Args) <= 1 {
		flaggy.ShowHelpAndExit("")
	}

	// URL
	if url == "" {
		printErrorAndExit("'--url(-u)' is required")
	}

	pgCfg, err := pgx.ParseConfig(url)

	if err != nil {
		printErrorAndExit("URL parsing error: " + err.Error())
	}

	flags.URL = url

	if pgCfg.Database == "" {
		pgCfg.Database = DefaultDBName
	}

	flags.PgConfig = &pgslap.PgConfig{
		ConnConfig: pgCfg,
		OnlyPrint:  flags.OnlyPrint,
	}

	// NAgents
	if flags.NAgents < 1 {
		printErrorAndExit("'--nagents(-n)' must be >= 1")
	}

	// NumberQueriesToExecute
	if flags.NumberQueriesToExecute < 0 {
		printErrorAndExit("'--number-queries' must be >= 0")
	}

	// Time
	if argTime < 0 {
		printErrorAndExit("'--time(-t)' must be >= 0")
	}

	flags.Time = time.Duration(argTime) * time.Second

	// Rate
	if flags.Rate < 0 {
		printErrorAndExit("'--rate(-r)' must be >= 0")
	}

	// Delimiter
	if delimiter == "" {
		printErrorAndExit("'--delimiter(-F)' must not be empty")
	}

	// AutoGenerateSql / Queries
	if !flags.AutoGenerateSql && queries == "" {
		printErrorAndExit("Either '--auto-generate-sql(-a)' or '--query(-q)' is required")
	} else if flags.AutoGenerateSql && queries != "" {
		printErrorAndExit("Cannot set both '--auto-generate-sql(-a)' and '--query(-q)'")
	}

	// Queries
	if queries != "" {
		if _, err := os.Stat(queries); err == nil {
			rawQueries, err := ioutil.ReadFile(queries)

			if err != nil {
				printErrorAndExit("Could not read the query file: " + queries)
			}

			queries = string(rawQueries)
		}

		flags.Queries = filterEmptyQuery(strings.Split(queries, delimiter))
	}

	// Creates
	if creates != "" {
		if queries == "" {
			printErrorAndExit("'--query(-q)' is required for '--create'")
		}

		if _, err := os.Stat(creates); err == nil {
			rawCreates, err := ioutil.ReadFile(creates)

			if err != nil {
				printErrorAndExit("Could not read the create SQL file: " + creates)
			}

			creates = string(rawCreates)
		}

		flags.Creates = filterEmptyQuery(strings.Split(creates, delimiter))
	}

	// NumberPrePopulatedData
	if flags.NumberPrePopulatedData < 0 {
		printErrorAndExit("'--auto-generate-sql-write-number' must be >= 0")
	}

	// LoadType
	loadType := pgslap.AutoGenerateSqlLoadType(strLoadType)

	if loadType != pgslap.LoadTypeMixed &&
		loadType != pgslap.LoadTypeUpdate &&
		loadType != pgslap.LoadTypeWrite &&
		loadType != pgslap.LoadTypeKey &&
		loadType != pgslap.LoadTypeRead {
		printErrorAndExit("Invalid load type: " + strLoadType)
	}

	if flags.NumberPrePopulatedData == 0 && (loadType == pgslap.LoadTypeMixed ||
		loadType == pgslap.LoadTypeUpdate ||
		loadType == pgslap.LoadTypeKey ||
		loadType == pgslap.LoadTypeRead) {
		printErrorAndExit("Pre-populated data is required for 'mixed', 'update', 'key', and 'read'")
	}

	flags.LoadType = loadType

	// NumberSecondaryIndexes
	if flags.NumberSecondaryIndexes < 0 {
		printErrorAndExit("'--auto-generate-sql-secondary-indexes' must be >= 0")
	}

	// CommitRate
	if flags.CommitRate < 0 {
		printErrorAndExit("'--commit-rate' must be >= 0")
	}

	// MixedSelRatio / MixedInsRatio
	if !strings.Contains(mixedSelInsRatio, ":") {
		printErrorAndExit("Invalid mixed type 'SELECT:INSERT' ratio: ':' is not included")
	}

	ratios := strings.SplitN(mixedSelInsRatio, ":", 2)
	flags.MixedSelRatio, err = strconv.Atoi(ratios[0])

	if err != nil {
		printErrorAndExit("Failed to parse SELECT ratio: " + err.Error())
	}

	if flags.MixedSelRatio < 1 {
		printErrorAndExit("Mixed type SELECT ratio must be >= 1")
	}

	flags.MixedInsRatio, err = strconv.Atoi(ratios[1])

	if err != nil {
		printErrorAndExit("Failed to parse INSERT ratio: " + err.Error())
	}

	if flags.MixedInsRatio < 1 {
		printErrorAndExit("Mixed type INSERT ratio must be >= 1")
	}

	// NumberIntCols
	if flags.NumberIntCols < 1 {
		printErrorAndExit("'--number-int-cols(-y)' must be >= 1")
	}

	// NumberCharCols
	if flags.NumberCharCols < 1 {
		printErrorAndExit("'--number-char-cols(-x)' must be >= 1")
	}

	// PreQueries
	if preqs != "" {
		flags.PreQueries = strings.Split(preqs, delimiter)
	}

	// HInterval
	if hi, err := time.ParseDuration(hinterval); err != nil {
		printErrorAndExit("Failed to parse hinterval: " + err.Error())
	} else {
		flags.HInterval = hi
	}

	return
}

func printErrorAndExit(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func filterEmptyQuery(queries []string) []string {
	filtered := []string{}

	for _, q := range queries {
		q = strings.TrimSpace(q)

		if q != "" {
			filtered = append(filtered, q)
		}
	}

	return filtered
}
