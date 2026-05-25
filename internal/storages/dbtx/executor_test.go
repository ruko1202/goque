package dbtx

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestWithTx_TxFromContext_Roundtrip(t *testing.T) {
	t.Parallel()

	tx := &sqlx.Tx{}
	ctx := WithTx(context.Background(), tx)

	gotTx, ok := TxFromContext(ctx)
	require.True(t, ok)
	require.Same(t, tx, gotTx)
}

func TestTxFromContext_Empty(t *testing.T) {
	t.Parallel()

	gotTx, ok := TxFromContext(context.Background())
	require.False(t, ok)
	require.Nil(t, gotTx)
}

func TestWithTx_ShadowsEarlierValue(t *testing.T) {
	t.Parallel()

	tx1, tx2 := &sqlx.Tx{}, &sqlx.Tx{}
	ctx := WithTx(context.Background(), tx1)
	ctx = WithTx(ctx, tx2)

	gotTx, ok := TxFromContext(ctx)
	require.True(t, ok)
	require.Same(t, tx2, gotTx)
}

func TestWithoutTx_RemovesAttachedTx(t *testing.T) {
	t.Parallel()

	tx := &sqlx.Tx{}
	ctx := WithTx(context.Background(), tx)
	ctx = WithoutTx(ctx)

	gotTx, ok := TxFromContext(ctx)
	require.False(t, ok)
	require.Nil(t, gotTx)
}

func TestWithoutTx_NoOpWhenAbsent(t *testing.T) {
	t.Parallel()

	base := context.Background()
	require.Equal(t, base, WithoutTx(base))
}

// TestWithTx_Nil pins the documented contract that WithTx(ctx, nil)
// is a no-op: the returned ctx is the input ctx (no value attached),
// and TxFromContext sees no tx. Without this, a caller who forgot to
// initialize their tx would silently get a plain *sqlx.DB write
// instead of the atomic outbox semantics they intended.
func TestWithTx_Nil(t *testing.T) {
	t.Parallel()

	base := context.Background()
	got := WithTx(base, nil)
	require.Equal(t, base, got, "WithTx(ctx, nil) must return the input ctx unchanged")
	gotTx, ok := TxFromContext(got)
	require.False(t, ok)
	require.Nil(t, gotTx)
}
