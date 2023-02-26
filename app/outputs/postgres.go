package outputs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pinebit/lognite/app/common"
	"github.com/pinebit/lognite/app/types"
	"go.uber.org/zap"
)

type Postgres interface {
	types.Service
	types.Output

	Connect(ctx context.Context, url string) error
	Close(ctx context.Context) error
	MigrateSchema(ctx context.Context, contracts types.ContractsPerChain) error
}

type postgres struct {
	db        *sqlx.DB
	logger    *zap.SugaredLogger
	queue     common.Queue[*types.Event]
	retention time.Duration
	lastPrune time.Time
}

var (
	errPostgresClosed = errors.New("postgres is closed")
)

func NewPostgres(logger *zap.SugaredLogger, retention time.Duration) Postgres {
	return &postgres{
		logger:    logger.Named("postgres"),
		queue:     common.NewQueue[*types.Event](uint(common.DefaultQueueCapacity)),
		retention: retention,
		lastPrune: time.Now().Add(-common.DefaultPostgresPruneInterval),
	}
}

func (d *postgres) Connect(ctx context.Context, url string) error {
	db, err := sqlx.Open("postgres", url)
	if err != nil {
		return err
	}
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	d.db = db
	return nil
}

func (d *postgres) Close(ctx context.Context) error {
	if d.db != nil {
		if err := d.db.Close(); err != nil {
			return err
		}
		d.db = nil
	}
	return nil
}

func (d postgres) Run(ctx context.Context, done func()) {
	defer done()

	d.queue.RunDequeueLoop(ctx, func(ctx context.Context, item *types.Event, discarded bool) {
		if discarded {
			common.PromQueueDiscarded.WithLabelValues("postgres").Inc()
			d.logger.Error("Postgres queue discarded one item")
		} else {
			d.handleEvent(ctx, item)
		}
	})
}

func (d postgres) MigrateSchema(ctx context.Context, contracts types.ContractsPerChain) error {
	if d.db == nil {
		return errPostgresClosed
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	for chainName, chainContracts := range contracts {
		_, err := tx.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS "+chainName)
		if err != nil {
			d.logger.Errorw("DB failed to create schema", "name", chainName, "err", err)
			defer tx.Rollback()
			return err
		}

		for _, contract := range chainContracts {
			tableName := eventsTableQN(contract)
			schema := `id BIGSERIAL PRIMARY KEY,
						ts TIMESTAMPTZ default now(),
						tx_hash TEXT NOT NULL,
						tx_index NUMERIC NOT NULL,
						block_number NUMERIC NOT NULL,
						address TEXT NOT NULL,
						event TEXT NOT NULL,
						args JSONB NOT NULL`
			q := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);", tableName, schema)
			_, err := tx.ExecContext(ctx, q)
			if err != nil {
				d.logger.Errorw("DB failed to create table", "err", err, "q", q)
				defer tx.Rollback()
				return err
			}

			columns := []string{"ts", "event"}
			for _, column := range columns {
				if err := d.createIndex(ctx, tx, contract, column); err != nil {
					d.logger.Errorw("DB failed to create index for column", "err", err, "tableName", tableName, "column", column)
					defer tx.Rollback()
					return err
				}
			}
		}
	}

	return tx.Commit()
}

func (d postgres) Write(event *types.Event) {
	d.queue.Enqueue(event)
}

func (d postgres) handleEvent(ctx context.Context, event *types.Event) {
	tableName := eventsTableQN(event.Contract)
	if event.Log.Removed {
		d.removeRecords(ctx, tableName, event.Log.BlockNumber)
	} else {
		d.insertRecord(ctx, tableName, event)
	}
	d.pruneEvents(ctx, tableName)
}

func (d postgres) createIndex(ctx context.Context, tx *sql.Tx, contract types.Contract, column string) error {
	q := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_%s_idx ON %s (%s);", contract.Name(), column, eventsTableQN(contract), column)
	_, err := tx.ExecContext(ctx, q)
	return err
}

func (d postgres) insertRecord(ctx context.Context, tableName string, event *types.Event) {
	jsonb, err := json.Marshal(event.Args)
	if err != nil {
		d.logger.Errorw("DB failed marshal jsonb", "err", err)
	} else {
		q := fmt.Sprintf("INSERT INTO %s (tx_hash, tx_index, block_number, address, event, args) VALUES ($1, $2, $3, $4, $5, $6)", tableName)
		_, err = d.db.ExecContext(ctx, q, event.Log.TxHash.Hex(), event.Log.TxIndex, event.Log.BlockNumber, event.Log.Address.Hex(), event.Name, jsonb)
		if err != nil {
			common.PromPostgresErrors.WithLabelValues(tableName).Inc()
			d.logger.Errorw("DB failed to insert", "err", err, "q", q)
		} else {
			common.PromPostgresInserts.WithLabelValues(tableName).Inc()
		}
	}
}

func (d postgres) removeRecords(ctx context.Context, tableName string, blockNumber uint64) {
	q := fmt.Sprintf("DELETE FROM %s WHERE block_number=%d", tableName, blockNumber)
	_, err := d.db.ExecContext(ctx, q)
	if err != nil {
		common.PromPostgresErrors.WithLabelValues(tableName).Inc()
		d.logger.Errorw("DB failed to drop", "err", err, "q", q)
	} else {
		common.PromPostgresDrops.WithLabelValues(tableName).Inc()
	}
}

func (d *postgres) pruneEvents(ctx context.Context, tableName string) {
	if time.Since(d.lastPrune) < common.DefaultPostgresPruneInterval {
		return
	}
	d.lastPrune = time.Now()
	deadline := time.Now().Add(-d.retention)
	q := fmt.Sprintf("DELETE FROM %s WHERE ts < $1;", tableName)
	_, err := d.db.ExecContext(ctx, q, deadline)
	if err != nil {
		d.logger.Errorw("DB failed to drop", "err", err, "q", q)
	}
}

func eventsTableQN(contract types.Contract) string {
	return fmt.Sprintf("%s.%s_events", contract.ChainName(), contract.Name())
}
