package sqlutil

import (
	"context"
	"database/sql"

	pkgerr "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/volatiletech/sqlboiler/boil"

	"github.com/Bnei-Baruch/gxydb-api/pkg/errs"
)

type TxError struct {
	errs.WithMessage
}

func WrappingTxError(err error, msg string) *TxError {
	return &TxError{errs.WithMessage{
		Msg: msg,
		Err: err,
	}}
}

func InTx(ctx context.Context, beginner boil.Beginner, f func(*sql.Tx) error) error {
	tx, err := beginner.Begin()
	if err != nil {
		return pkgerr.WithStack(WrappingTxError(err, "begin tx"))
	}

	// rollback on panics
	defer func() {
		if p := recover(); p != nil {
			if ex := tx.Rollback(); ex != nil {
				log.Ctx(ctx).Error().Err(ex).Msg("rollback error on panic")
			}
			panic(p) // re-throw panic after Rollback
		}
	}()

	// invoke logic and rollback on errors
	if err := f(tx); err != nil {
		if ex := tx.Rollback(); ex != nil {
			return pkgerr.WithStack(WrappingTxError(err, "tx.Rollback"))
		}
		return err
	}

	// commit transaction
	if err := tx.Commit(); err != nil {
		return pkgerr.WithStack(WrappingTxError(err, "tx.Commit"))
	}

	return nil
}
