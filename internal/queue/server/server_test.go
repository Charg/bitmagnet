package server

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bitmagnet-io/bitmagnet/internal/database/dao"
	"github.com/bitmagnet-io/bitmagnet/internal/database/postgrestest"
	"github.com/bitmagnet-io/bitmagnet/internal/model"
	"github.com/bitmagnet-io/bitmagnet/internal/queue/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestHandler(t *testing.T, q *dao.Query, f handler.Func) (*serverHandler, string) {
	t.Helper()

	queueName := fmt.Sprintf("test_%d", time.Now().UnixNano())

	t.Cleanup(func() {
		_, _ = q.QueueJob.WithContext(context.Background()).Where(q.QueueJob.Queue.Eq(queueName)).Delete()
	})

	return &serverHandler{
		Handler: handler.New(queueName, f, handler.JobTimeout(time.Minute)),
		query:   q,
		logger:  zap.NewNop().Sugar(),
	}, queueName
}

func insertJob(t *testing.T, q *dao.Query, queueName string, opts ...model.QueueJobOption) model.QueueJob {
	t.Helper()

	job, err := model.NewQueueJob(queueName, map[string]any{"n": time.Now().UnixNano()}, opts...)
	require.NoError(t, err)
	require.NoError(t, q.QueueJob.WithContext(context.Background()).Create(&job))

	return job
}

func getJob(ctx context.Context, t *testing.T, q *dao.Query, id string) *model.QueueJob {
	t.Helper()

	job, err := q.QueueJob.WithContext(ctx).Where(q.QueueJob.ID.Eq(id)).First()
	require.NoError(t, err)

	return job
}

func TestHandleJobRunsHandlerOutsideTransactionUnderLease(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	q := postgrestest.Open(t)

	var (
		h         *serverHandler
		queueName string
		job       model.QueueJob
	)

	h, queueName = newTestHandler(t, q, func(ctx context.Context, _ model.QueueJob) error {
		// while the job runs, it is not held in an open transaction (so a concurrent update doesn't block)...
		updateCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		_, err := q.QueueJob.WithContext(updateCtx).Where(q.QueueJob.ID.Eq(job.ID)).
			UpdateSimple(q.QueueJob.Priority.Value(job.Priority))
		require.NoError(t, err, "the job row should not be locked while the handler runs")

		// ...and another worker can't claim it, because of the lease
		other, _, claimErr := h.claimJob(ctx)
		require.NoError(t, claimErr)
		assert.Nil(t, other, "a leased job should not be claimable")

		leased := getJob(ctx, t, q, job.ID)
		assert.Equal(t, model.QueueJobStatusPending, leased.Status)
		assert.True(
			t,
			leased.RunAfter.After(time.Now().Add(time.Minute)),
			"the lease should cover the job timeout",
		)

		return nil
	})

	job = insertJob(t, q, queueName)

	jobID, processed, err := h.handleJob(ctx)
	require.NoError(t, err)
	assert.Equal(t, job.ID, jobID)
	assert.True(t, processed)

	done := getJob(ctx, t, q, job.ID)
	assert.Equal(t, model.QueueJobStatusProcessed, done.Status)
	assert.True(t, done.RanAt.Valid)
	assert.WithinDuration(t, job.RunAfter, done.RunAfter, time.Millisecond,
		"the lease should not be left on the processed job")

	// nothing left to claim:
	jobID, processed, err = h.handleJob(ctx)
	require.NoError(t, err)
	assert.Empty(t, jobID)
	assert.False(t, processed)
}

func TestHandleJobRecordsFailureForRetry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	q := postgrestest.Open(t)

	h, queueName := newTestHandler(t, q, func(context.Context, model.QueueJob) error {
		return errors.New("boom")
	})

	job := insertJob(t, q, queueName, model.QueueJobMaxRetries(2))

	_, processed, err := h.handleJob(ctx)
	require.NoError(t, err)
	assert.False(t, processed)

	failed := getJob(ctx, t, q, job.ID)
	assert.Equal(t, model.QueueJobStatusRetry, failed.Status)
	assert.Contains(t, failed.Error.String, "boom")
	assert.True(t, failed.RunAfter.After(time.Now()), "a retry should be backed off")
	assert.True(
		t,
		failed.RunAfter.Before(time.Now().Add(time.Minute)),
		"the lease should be replaced by the backoff",
	)
}

func TestHandleJobDoesNotOverwriteAfterLosingLease(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	q := postgrestest.Open(t)

	var job model.QueueJob

	h, queueName := newTestHandler(t, q, func(ctx context.Context, _ model.QueueJob) error {
		// simulate the lease expiring and another worker claiming the job:
		_, err := q.QueueJob.WithContext(ctx).Where(q.QueueJob.ID.Eq(job.ID)).
			UpdateSimple(q.QueueJob.RunAfter.Value(time.Now().Add(time.Hour)))

		return err
	})

	job = insertJob(t, q, queueName)

	_, processed, err := h.handleJob(ctx)
	require.NoError(t, err)
	assert.False(t, processed)

	assert.Equal(t, model.QueueJobStatusPending, getJob(ctx, t, q, job.ID).Status,
		"the outcome should not be written once the lease is lost")
}
