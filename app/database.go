package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type Database interface {
	Service
	Output

	Connect(ctx context.Context, url string) error
	Close(ctx context.Context) error
	MigrateSchema(ctx context.Context, contracts map[string][]Contract) error
}

type database struct {
	db        *sqlx.DB
	logger    *zap.SugaredLogger
	queue     chan *Event
	retention time.Duration
	lastPrune time.Time
}

var (
	errDatabaseClosed = errors.New("database is closed")
)

func NewDatabase(logger *zap.SugaredLogger, config *PostgresConfig) Database {
	return &database{
		logger:    logger.Named("database"),
		queue:     make(chan *Event, defaultOutputBuffer),
		retention: *config.Retention,
		lastPrune: time.Now().Add(-defaultPostgresPruneInterval),
	}
}

func (d *database) Connect(ctx context.Context, url string) error {
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

func (d *database) Close(ctx context.Context) error {
	if d.db != nil {
		close(d.queue)
		if err := d.db.Close(); err != nil {
			return err
		}
		d.db = nil
	}
	return nil
}

func (d database) Run(ctx context.Context) (err error) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-d.queue:
			if !ok {
				return
			}
			d.handleEvent(ctx, event)
		}
	}
}

func (d database) MigrateSchema(ctx context.Context, contracts map[string][]Contract) error {
	if d.db == nil {
		return errDatabaseClosed
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
						ts TIMESTAMP WITHOUT TIME ZONE default (now() at time zone 'utc'),
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

			columns := []string{"ts", "tx_hash", "block_number", "address", "event"}
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

func (d database) Write(event *Event) {
	d.queue <- event
}

func (d database) handleEvent(ctx context.Context, event *Event) {
	tableName := eventsTableQN(event.Contract)
	if event.Log.Removed {
		d.removeRecords(ctx, tableName, event.Log.BlockNumber)
	} else {
		d.insertRecord(ctx, tableName, event)
	}
	d.pruneEvents(ctx, tableName)
}

func (d database) createIndex(ctx context.Context, tx *sql.Tx, contract Contract, column string) error {
	q := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_%s_idx ON %s (%s);", contract.Name(), column, eventsTableQN(contract), column)
	_, err := tx.ExecContext(ctx, q)
	return err
}

func (d database) insertRecord(ctx context.Context, tableName string, event *Event) {
	jsonb, err := json.Marshal(event.Args)
	if err != nil {
		d.logger.Errorw("DB failed marshal jsonb", "err", err)
	} else {
		q := fmt.Sprintf("INSERT INTO %s (tx_hash, tx_index, block_number, address, event, args) VALUES ($1, $2, $3, $4, $5, $6)", tableName)
		_, err = d.db.ExecContext(ctx, q, event.Log.TxHash.Hex(), event.Log.TxIndex, event.Log.BlockNumber, event.Log.Address.Hex(), event.Name, jsonb)
		if err != nil {
			promDBErrors.WithLabelValues(tableName).Inc()
			d.logger.Errorw("DB failed to insert", "err", err, "q", q)
		} else {
			promDBInserts.WithLabelValues(tableName).Inc()
		}
	}
}

func (d database) removeRecords(ctx context.Context, tableName string, blockNumber uint64) {
	q := fmt.Sprintf("DELETE FROM %s WHERE block_number=%d", tableName, blockNumber)
	_, err := d.db.ExecContext(ctx, q)
	if err != nil {
		promDBErrors.WithLabelValues(tableName).Inc()
		d.logger.Errorw("DB failed to drop", "err", err, "q", q)
	} else {
		promDBDrops.WithLabelValues(tableName).Inc()
	}
}

func (d *database) pruneEvents(ctx context.Context, tableName string) {
	if time.Since(d.lastPrune) < defaultPostgresPruneInterval {
		return
	}
	d.lastPrune = time.Now()
	hours := uint(d.retention.Hours())
	q := fmt.Sprintf("DELETE FROM %s WHERE ts < (now() at time zone 'utc') - '%d hours'::interval;", tableName, hours)
	_, err := d.db.ExecContext(ctx, q)
	if err != nil {
		d.logger.Errorw("DB failed to drop", "err", err, "q", q)
	}
}

func eventsTableQN(contract Contract) string {
	return fmt.Sprintf("%s.%s_events", contract.ChainName(), contract.Name())
}
