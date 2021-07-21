package pgslap

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

const (
	RecordPeriod = 1 * time.Second
)

type Agent struct {
	id       int
	pgConfig *PgConfig
	db       DB
	taskOps  *TaskOpts
	dataOpts *DataOpts
	data     *Data
}

func newAgent(id int, pgCfg *PgConfig, taskOps *TaskOpts, dataOpts *DataOpts) (agent *Agent) {
	agent = &Agent{
		id:       id,
		pgConfig: pgCfg,
		taskOps:  taskOps,
		dataOpts: dataOpts,
	}

	return
}

func (agent *Agent) prepare(idList []string) error {
	conn, err := agent.pgConfig.openAndPing()

	if err != nil {
		dsn := agent.pgConfig.ConnString()
		return fmt.Errorf("Failed to open/ping DB (agent id=%d, dsn=%s): %w", agent.id, dsn, err)
	}

	agent.db = conn
	newIdList := make([]string, len(idList))
	copy(newIdList, idList)
	rand.Shuffle(len(newIdList), func(i, j int) { newIdList[i], newIdList[j] = newIdList[j], newIdList[i] })
	agent.data = newData(agent.dataOpts, newIdList)

	inits := agent.data.initStmts()

	for _, stmt := range inits {
		_, err = conn.Exec(context.Background(), stmt)

		if err != nil {
			return fmt.Errorf("Failed to execute initial query (agent id=%d, query=%s): %w", agent.id, stmt, err)
		}
	}

	return nil
}

func (agent *Agent) run(ctx context.Context, recorder *Recorder) error {
	recordTick := time.NewTicker(RecordPeriod)
	defer recordTick.Stop()
	recDps := []recorderDataPoint{}

	err := loopWithThrottle(agent.taskOps.Rate, func(i int) (bool, error) {
		if agent.taskOps.NumberQueriesToExecute > 0 && i >= agent.taskOps.NumberQueriesToExecute {
			return false, nil
		}

		select {
		case <-ctx.Done():
			return false, nil
		case <-recordTick.C:
			recorder.add(recDps)
			recDps = recDps[:0]
		default:
			// Nothing to do
		}

		q, args := agent.data.next()
		rt, err := agent.query(ctx, q, args...)

		if err != nil {
			return false, fmt.Errorf("Execute query error (query=%s, args=%v): %w", q, args, err)
		}

		recDps = append(recDps, recorderDataPoint{
			timestamp: time.Now(),
			resTime:   rt,
		})

		return true, nil
	})

	if err != nil {
		return fmt.Errorf("Failed to transact (agent id=%d): %w", agent.id, err)
	}

	return nil
}

func (agent *Agent) close() error {
	err := agent.db.Close(context.Background())

	if err != nil {
		return fmt.Errorf("Failed to close DB (agent id=%d): %w", agent.id, err)
	}

	return nil
}

func (agent *Agent) query(ctx context.Context, q string, args ...interface{}) (time.Duration, error) {
	start := time.Now()
	_, err := agent.db.Exec(ctx, q, args...)
	end := time.Now()

	if err != nil && !errors.Is(err, context.Canceled) && !pgconn.Timeout(err) {
		// NOTE: Connection may close due to timeout..
		// cf.
		// * https://github.com/jackc/pgconn/blob/a50d96d4915cae7d1a28601ce9e7a57b0ea5ae41/errors.go#L20-L21
		// * https://github.com/jackc/pgconn/issues/81
		if pgxConn, ok := agent.db.(*pgx.Conn); !ok || !pgxConn.IsClosed() {
			return 0, err
		}
	}

	return end.Sub(start), nil
}
