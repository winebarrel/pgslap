# pgslap

pgslap is a PostgreSQL load testing tool like [mysqlslap](https://dev.mysql.com/doc/refman/8.0/en/mysqlslap.html).

## Usage

```
pgslap - PostgreSQL load testing tool like mysqlslap.

  Flags:
       --version                               Displays the program version string.
    -h --help                                  Displays help with available flag, subcommand, and positional value parameters.
    -u --url                                   Database URL, e.g. 'postgres://username:password@localhost:5432'.
    -n --nagents                               Number of agents. (default: 1)
    -t --time                                  Test run time (sec). Zero is infinity. (default: 60)
       --number-queries                        Number of queries to execute per agent. Zero is infinity. (default: 0)
    -r --rate                                  Rate limit for each agent (qps). Zero is unlimited. (default: 0)
    -a --auto-generate-sql                     Automatically generate SQL to execute.
       --auto-generate-sql-guid-primary        Use GUID as the primary key of the table to be created.
    -q --query                                 SQL to execute. (file or string)
       --auto-generate-sql-write-number        Number of rows to be pre-populated for each agent. (default: 100)
    -l --auto-generate-sql-load-type           Test load type: 'mixed', 'update', 'write', 'key', or 'read'. (default: mixed)
       --auto-generate-sql-secondary-indexes   Number of secondary indexes in the table to be created. (default: 0)
       --commit-rate                           Commit every X queries. (default: 0)
       --mixed-sel-ins-ratio                   Mixed load type 'SELECT:INSERT' ratio. (default: 1:1)
    -x --number-char-cols                      Number of VARCHAR columns in the table to be created. (default: 1)
       --char-cols-index                       Create indexes on VARCHAR columns in the table to be created.
    -y --number-int-cols                       Number of INT columns in the table to be created. (default: 1)
       --int-cols-index                        Create indexes on INT columns in the table to be created.
       --pre-query                             Queries to be pre-executed for each agent.
       --create                                SQL for creating custom tables. (file or string)
       --drop-db                               Forcibly delete the existing DB.
       --no-drop                               Do not drop database after testing.
       --hinterval                             Histogram interval, e.g. '100ms'. (default: 0)
    -F --delimiter                             SQL statements delimiter. (default: ;)
       --only-print                            Just print SQL without connecting to DB.
       --no-progress                           Do not show progress.
```

```
$ pgslap -u 'postgres://scott@localhost:5432' -n 10 -r 100 -t 10 -a -l mixed -x 3 -y 3
00:10 | 10 agents / run 9090 queries (1010 qps)

{
  "URL": "postgres://scott@localhost:5432",
  "StartedAt": "2021-07-21T17:08:10.919264+09:00",
  "FinishedAt": "2021-07-21T17:08:20.932804+09:00",
  "ElapsedTime": 10,
  "NAgents": 10,
  "Rate": 100,
  "AutoGenerateSql": true,
  "NumberPrePopulatedData": 100,
  "NumberQueriesToExecute": 0,
  "DropExistingDatabase": false,
  "UseExistingDatabase": true,
  "NoDropDatabase": false,
  "LoadType": "mixed",
  "GuidPrimary": false,
  "NumberSecondaryIndexes": 0,
  "CommitRate": 0,
  "MixedSelRatio": 1,
  "MixedInsRatio": 1,
  "NumberIntCols": 3,
  "IntColsIndex": false,
  "NumberCharCols": 3,
  "CharColsIndex": false,
  "PreQueries": null,
  "GOMAXPROCS": 16,
  "QueryCount": 9595,
  "AvgQPS": 958.2145909634482,
  "MaxQPS": 1010,
  "MinQPS": 5,
  "MedianQPS": 1010,
  "ExpectedQPS": 1000,
  "Response": {
    "Time": {
      "Cumulative": "4.808905566s",
      "HMean": "340.992µs",
      "Avg": "501.188µs",
      "P50": "404.475µs",
      "P75": "687.146µs",
      "P95": "902.047µs",
      "P99": "1.25769ms",
      "P999": "6.284183ms",
      "Long5p": "1.451295ms",
      "Short5p": "164.521µs",
      "Max": "6.708257ms",
      "Min": "3.04µs",
      "Range": "6.705217ms",
      "StdDev": "416.557µs"
    },
    "Rate": {
      "Second": 1995.2564816075244
    },
    "Samples": 9595,
    "Count": 9595,
    "Histogram": [
      {
        "3µs - 673µs": 7044
      },
      {
        "673µs - 1.344ms": 2465
      },
      {
        "1.344ms - 2.014ms": 20
      },
      {
        "2.014ms - 2.685ms": 16
      },
      {
        "2.685ms - 3.355ms": 4
      },
      {
        "3.355ms - 4.026ms": 14
      },
      {
        "4.026ms - 4.696ms": 12
      },
      {
        "4.696ms - 5.367ms": 2
      },
      {
        "5.367ms - 6.037ms": 3
      },
      {
        "6.037ms - 6.708ms": 15
      }
    ]
  }
}
```

## Use Custom Query

```
pgslap -u 'postgres://scott@localhost:5432' \
  --create 'create table test (id int); insert into test values (1)' \
  -q 'select id from test; select count(id) from test'
```

## Related Links

* MySQL load testing tool like mysqlslap
    * https://github.com/winebarrel/qlap
