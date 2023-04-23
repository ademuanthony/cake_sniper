// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/volatiletech/sqlboiler/v4/boil"
)

const dateTemplate = "2006-01-02 15:04"
const dateMiliTemplate = "2006-01-02 15:04:05.99"

var (
	// PGCancelError is the error string PostgreSQL returns when a query fails
	// to complete due to user requested cancellation.
	PGCancelError       = "pq: canceling statement due to user request"
	CtxDeadlineExceeded = context.DeadlineExceeded.Error()
	TimeoutPrefix       = "TIMEOUT of PostgreSQL query"
)

const DateTemplate = "2006-01-02 15:04"
const DateMiliTemplate = "2006-01-02 15:04:05.99"

type insertable interface {
	Insert(context.Context, boil.ContextExecutor, boil.Columns) error
}

type upsertable interface {
	Upsert(ctx context.Context, db boil.ContextExecutor, updateOnConflict bool, conflictColumns []string, updateColumns, insertColumns boil.Columns) error
}

func TableExists(db *sql.DB, name string) (bool, error) {
	rows, err := db.Query(`SELECT relname FROM pg_class WHERE relname = $1`, name)
	if err == nil {
		defer func() {
			if e := rows.Close(); e != nil {
				fmt.Println("Close of Query failed: ", e)
			}
		}()
		return rows.Next(), nil
	}
	return false, err
}

func DropTable(db *sql.DB, name string) error {
	fmt.Println("Dropping table ", name)
	_, err := db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s;`, name))
	return err
}

// IsTimeout checks if the message is prefixed with the expected DB timeout
// message prefix.
func IsTimeout(msg string) bool {
	// Contains is used instead of HasPrefix since error messages are often
	// supplemented with additional information.
	return strings.Contains(msg, TimeoutPrefix) ||
		strings.Contains(msg, CtxDeadlineExceeded)
}

// IsTimeoutErr checks if error's message is prefixed with the expected DB
// timeout message prefix.
func IsTimeoutErr(err error) bool {
	return err != nil && IsTimeout(err.Error())
}

func RoundValue(input float64) string {
	value := input * 100
	return strconv.FormatFloat(value, 'f', 3, 64)
}

func (pg *PgDb) tryInsert(ctx context.Context, txr boil.Transactor, data insertable) error {
	err := data.Insert(ctx, pg.Db, boil.Infer())
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			return err
		}
		errT := txr.Rollback()
		if errT != nil {
			return errT
		}
		return err
	}
	return nil
}

func (pg *PgDb) tryUpsert(ctx context.Context, txr boil.Transactor, data upsertable) error {
	err := data.Upsert(ctx, pg.Db, true, nil, boil.Infer(), boil.Infer())
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			return err
		}
		errT := txr.Rollback()
		if errT != nil {
			return errT
		}
		return err
	}
	return nil
}

func isUniqueConstraint(err error) bool {
	return err != nil && strings.Contains(err.Error(), "unique constraint")
}
