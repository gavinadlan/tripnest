// Package pgtx membawa transaksi Postgres lewat context.
//
// Repository memanggil From(ctx, pool) untuk setiap query: kalau ctx
// sedang membawa transaksi, query ikut di dalamnya; kalau tidak, query
// jalan langsung di pool. Dengan begitu satu method repository yang sama
// bisa dipakai sendirian (lewat HTTP) maupun sebagai bagian dari
// transaksi yang lebih besar (consumer Kafka yang harus menulis efek
// bisnis dan catatan idempotensi bersama-sama), tanpa dua versi method.
package pgtx

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier adalah irisan pgxpool.Pool dan pgx.Tx yang dipakai repository.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type ctxKey struct{}

// From mengembalikan transaksi yang dibawa ctx, atau pool bila tidak ada.
func From(ctx context.Context, pool *pgxpool.Pool) Querier {
	if tx, ok := ctx.Value(ctxKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}

// InTx melaporkan apakah ctx sedang membawa transaksi.
func InTx(ctx context.Context) bool {
	_, ok := ctx.Value(ctxKey{}).(pgx.Tx)
	return ok
}

// Transact menjalankan fn di dalam satu transaksi. fn menerima ctx baru
// yang membawa transaksinya; commit bila fn mengembalikan nil, rollback
// bila tidak.
//
// Kalau ctx sudah membawa transaksi, fn dijalankan di dalamnya tanpa
// membuka transaksi baru. Ini yang membuat method repository yang sudah
// transaksional (misalnya ReserveSlot) tetap bisa dibungkus lagi oleh
// consumer tanpa menghasilkan dua transaksi terpisah.
func Transact(ctx context.Context, pool *pgxpool.Pool, fn func(ctx context.Context) error) (err error) {
	if InTx(ctx) {
		return fn(ctx)
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("gagal membuka transaksi: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				err = errors.Join(err, rbErr)
			}
			return
		}
		if cErr := tx.Commit(ctx); cErr != nil {
			err = fmt.Errorf("gagal commit transaksi: %w", cErr)
		}
	}()

	return fn(context.WithValue(ctx, ctxKey{}, tx))
}

// NewPool membuka pool koneksi dari connection string. Dipisah dari
// repository supaya pool yang sama bisa dipakai worker outbox dan loop
// consumer tanpa membuka koneksi kedua.
func NewPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("konfigurasi database tidak valid: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka koneksi database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database tidak bisa dihubungi: %w", err)
	}
	return pool, nil
}
