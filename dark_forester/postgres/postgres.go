// Copyright (c) 2018-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package postgres

//go:generate sqlboiler --wipe psql --no-hooks --no-auto-timestamps

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/volatiletech/sqlboiler/v4/boil"
)

type PgDb struct {
	Db           *sql.DB
	queryTimeout time.Duration
}

type pgLogWriter struct{}

func (l pgLogWriter) Write(p []byte) (n int, err error) {
	log.Info(string(p))
	return len(p), nil
}

func NewPgDb(host, port, user, pass, dbname string, debug bool) (*PgDb, error) {
	db, err := Connect(host, port, user, pass, dbname)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(5)
	if debug {
		boil.DebugMode = true
		boil.DebugWriter = pgLogWriter{}
	}
	return &PgDb{
		Db:           db,
		queryTimeout: time.Second * 30,
	}, nil
}

func (pg *PgDb) Close() error {
	log.Info("Closing postgresql connection")
	return pg.Db.Close()
}

func (pg *PgDb) BeginTransaction() (*sql.Tx, error) {
	return pg.Db.Begin()
}

func (pg *PgDb) timeoutError() string {
	return fmt.Sprintf("%s after %v", TimeoutPrefix, pg.queryTimeout)
}

// replaceCancelError will replace the generic error strings that can occur when
// a PG query is canceled (dbtypes.PGCancelError) or a context deadline is
// exceeded (dbtypes.CtxDeadlineExceeded from context.DeadlineExceeded).
func (pg *PgDb) replaceCancelError(err error) error {
	if err == nil {
		return err
	}

	patched := err.Error()
	if strings.Contains(patched, PGCancelError) {
		patched = strings.Replace(patched, PGCancelError,
			pg.timeoutError(), -1)
	} else if strings.Contains(patched, CtxDeadlineExceeded) {
		patched = strings.Replace(patched, CtxDeadlineExceeded,
			pg.timeoutError(), -1)
	} else {
		return err
	}
	return errors.New(patched)
}
