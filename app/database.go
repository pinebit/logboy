package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

type Database interface {
	Output

	Connect(ctx context.Context, url string) error
	Close(ctx context.Context) error
	MigrateSchema(ctx context.Context, rpcs []Chain) error
}

type database struct {
	db     *sqlx.DB
	logger *zap.SugaredLogger
}

var (
	errDatabaseClosed = errors.New("database is closed")

	promDBErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_db_errors",
		Help: "The total number of DB errors per table",
	}, []string{"table"})

	promDBInserts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_db_inserts",
		Help: "The total number of DB inserts per table",
	}, []string{"table"})

	promDBDrops = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lognite_db_drops",
		Help: "The total number of DB drops per table",
	}, []string{"table"})
)

func NewDatabase(logger *zap.SugaredLogger) Database {
	return &database{
		logger: logger,
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
		if err := d.db.Close(); err != nil {
			return err
		}
		d.db = nil
	}
	return nil
}

func (d database) MigrateSchema(ctx context.Context, chains []Chain) error {
	if d.db == nil {
		return errDatabaseClosed
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	for _, chain := range chains {
		_, err := tx.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS "+chain.Name())
		if err != nil {
			d.logger.Errorw("DB failed to create schema", "name", chain.Name(), "err", err)
			defer tx.Rollback()
			return err
		}

		for _, contract := range chain.Contracts() {
			columns := `id BIGSERIAL PRIMARY KEY,
						ts TIMESTAMP WITHOUT TIME ZONE default (now() at time zone 'utc'),
						tx_hash TEXT NOT NULL,
						tx_index NUMERIC NOT NULL,
						block_number NUMERIC NOT NULL,
						address TEXT NOT NULL,
						event TEXT NOT NULL,
						args JSONB NOT NULL`
			q := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);", eventsTableQN(contract), columns)
			_, err := tx.ExecContext(ctx, q)
			if err != nil {
				d.logger.Errorw("DB failed to create table", "err", err, "q", q)
				defer tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

func (d database) Write(ctx context.Context, log types.Log, contract Contract, event string, args map[string]interface{}) {
	tableName := eventsTableQN(contract)
	if log.Removed {
		d.removeRecords(ctx, tableName, log.TxHash)
	} else {
		d.insertRecord(ctx, tableName, log, event, args)
	}
}

func (d database) insertRecord(ctx context.Context, tableName string, log types.Log, event string, args map[string]interface{}) {
	jsonb, err := json.Marshal(args)
	if err != nil {
		d.logger.Errorw("DB failed marshal jsonb", "err", err)
	} else {
		q := fmt.Sprintf("INSERT INTO %s (tx_hash, tx_index, block_number, address, event, args) VALUES ($1, $2, $3, $4, $5, $6)", tableName)
		_, err = d.db.ExecContext(ctx, q, log.TxHash.Hex(), log.TxIndex, log.BlockNumber, log.Address.Hex(), event, jsonb)
		if err != nil {
			promDBErrors.WithLabelValues(tableName).Inc()
			d.logger.Errorw("DB failed to insert", "err", err, "q", q)
		} else {
			promDBInserts.WithLabelValues(tableName).Inc()
		}
	}
}

func (d database) removeRecords(ctx context.Context, tableName string, txHash common.Hash) {
	q := fmt.Sprintf("DROP FROM %s WHERE tx_hash=%s", tableName, txHash.Hex())
	_, err := d.db.ExecContext(ctx, q)
	if err != nil {
		promDBErrors.WithLabelValues(tableName).Inc()
		d.logger.Errorw("DB failed to drop", "err", err, "q", q)
	} else {
		promDBDrops.WithLabelValues(tableName).Inc()
	}
}

func eventsTableQN(contract Contract) string {
	return fmt.Sprintf("%s.%s_events", contract.Chain(), contract.Name())
}
