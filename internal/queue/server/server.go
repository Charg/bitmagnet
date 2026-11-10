// the listener connection code has been disabled:
// it would rarely be used anyway since a delay is now added to crawler jobs;
// if re-enabled in the future, some work is needed to gracefully handle disconnection

package server

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bitmagnet-io/bitmagnet/internal/database/dao"
	"github.com/bitmagnet-io/bitmagnet/internal/model"
	"github.com/bitmagnet-io/bitmagnet/internal/queue"
	"github.com/bitmagnet-io/bitmagnet/internal/queue/handler"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
	"gorm.io/gen"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type server struct {
	stopped chan struct{}
	query   *dao.Query
	// pool       *pgxpool.Pool
	handlers   []handler.Handler
	gcInterval time.Duration
	logger     *zap.SugaredLogger
}

func (s *server) Start(ctx context.Context) (err error) {
	ctx, cancel := context.WithCancel(ctx)

	defer func() {
		if err != nil {
			cancel()
		}
	}()
	// pListenerConn, listenerConnErr := s.newListenerConn(ctx)
	// if listenerConnErr != nil {
	// 	err = listenerConnErr
	// 	return
	// }
	// listenerConn := pListenerConn.Conn()
	handlers := make([]serverHandler, len(s.handlers))
	listenerChans := make(map[string]chan pgconn.Notification)

	for i, h := range s.handlers {
		listenerChan := make(chan pgconn.Notification)
		sh := serverHandler{
			Handler: h,
			sem:     semaphore.NewWeighted(int64(h.Concurrency)),
			query:   s.query,
			// listenerConn: listenerConn,
			listenerChan: listenerChan,
			logger:       s.logger.With("queue", h.Queue),
		}
		handlers[i] = sh
		listenerChans[h.Queue] = listenerChan
		// if _, listenErr := listenerConn.Exec(ctx, fmt.Sprintf(`LISTEN %q`, h.Queue)); listenErr != nil {
		//	err = listenErr
		//	return
		//}
		go sh.start(ctx)
	}

	go func() {
		for {
			select {
			case <-s.stopped:
				cancel()
			case <-ctx.Done():
				// pListenerConn.Release()
				return
			}
		}
	}()
	// go func() {
	// 	for {
	// 		select {
	// 		case <-ctx.Done():
	// 			return
	// 		default:
	// 			notification, waitErr := listenerConn.WaitForNotification(ctx)
	// 			if waitErr != nil {
	// 				if !errors.Is(waitErr, context.Canceled) {
	// 					s.logger.Errorf("Error waiting for notification: %s", waitErr)
	// 				}
	// 				continue
	// 			}
	// 			ch, ok := listenerChans[notification.Channel]
	// 			if !ok {
	// 				s.logger.Errorf("Received notification for unknown channel: %s", notification.Channel)
	// 				continue
	// 			}
	// 			select {
	// 			case <-ctx.Done():
	// 				return
	// 			case ch <- *notification:
	// 				continue
	// 			}
	// 		}
	// 	}
	// }()
	go s.runGarbageCollection(ctx)

	return
}

// func (s *server) newListenerConn(ctx context.Context) (*pgxpool.Conn, error) {
//	conn, err := s.pool.Acquire(ctx)
//	if err != nil {
//		return nil, err
//	}
//	_, err = conn.Exec(ctx, "SET idle_in_transaction_session_timeout = 0")
//	if err != nil {
//		return nil, err
//	}
//	return conn, nil
//}

func (s *server) runGarbageCollection(ctx context.Context) {
	for {
		tx := s.query.QueueJob.WithContext(ctx).Where(
			s.query.QueueJob.Status.In(string(model.QueueJobStatusProcessed), string(model.QueueJobStatusFailed)),
		).
			UnderlyingDB().Where(
			"queue_jobs.ran_at + queue_jobs.archival_duration < ?::timestamptz",
			time.Now(),
		).Delete(&model.QueueJob{})
		if tx.Error != nil {
			s.logger.Errorw("error deleting old queue jobs", "error", tx.Error)
		} else if tx.RowsAffected > 0 {
			s.logger.Debugw("deleted old queue jobs", "count", tx.RowsAffected)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(s.gcInterval):
			continue
		}
	}
}

type serverHandler struct {
	handler.Handler
	sem   *semaphore.Weighted
	query *dao.Query
	// listenerConn *pgx.Conn
	listenerChan chan pgconn.Notification
	logger       *zap.SugaredLogger
}

func (h *serverHandler) start(ctx context.Context) {
	checkTicker := time.NewTicker(1)

	for {
		select {
		case <-ctx.Done():
			return
		case notification := <-h.listenerChan:
			if semErr := h.sem.Acquire(ctx, 1); semErr != nil {
				return
			}

			go func() {
				defer h.sem.Release(1)
				_, _, _ = h.handleJob(ctx, h.query.QueueJob.ID.Eq(notification.Payload))
			}()
		case <-checkTicker.C:
			if semErr := h.sem.Acquire(ctx, 1); semErr != nil {
				return
			}

			checkTicker.Reset(h.CheckInterval)

			go func() {
				defer h.sem.Release(1)
				jobID, _, err := h.handleJob(ctx)
				// if a job was found, we should check straight away for another job,
				// otherwise we wait for the check interval
				if err == nil && jobID != "" {
					checkTicker.Reset(1)
				}
			}()
		}
	}
}

// leaseGrace is added to a handler's job timeout to give the lease on a claimed job; if the process dies
// mid-job, the job becomes claimable again once the lease expires.
const leaseGrace = time.Minute

// resultWriteTimeout bounds the write of a job's outcome, which must happen even if the server is stopping.
const resultWriteTimeout = 30 * time.Second

// handleJob claims a job, executes it, and records the outcome.
//
// The job is claimed in a short transaction that takes a lease on it by pushing its run_after past the job
// timeout; the handler itself runs outside any transaction. Holding a transaction open for the duration of a job
// (which can include slow external requests) pins a connection and holds back vacuum for every table.
func (h *serverHandler) handleJob(
	ctx context.Context,
	conds ...gen.Condition,
) (jobID string, processed bool, err error) {
	job, leaseUntil, claimErr := h.claimJob(ctx, conds...)
	if claimErr != nil {
		h.logger.Errorw("error claiming job", "error", claimErr)

		return "", false, claimErr
	}

	if job == nil {
		return "", false, nil
	}

	jobID = job.ID

	var jobErr error
	if job.Deadline.Valid && job.Deadline.Time.Before(time.Now()) {
		jobErr = ErrJobExceededDeadline

		h.logger.Debugw("job deadline is in the past, skipping", "job_id", job.ID)
	} else {
		// check if the job is being retried and increment retry count accordingly
		if job.Status != model.QueueJobStatusPending {
			job.Retries++
		}
		// execute the queue handler of this job
		jobErr = handler.Exec(ctx, h.Handler, *job)
	}

	job.RanAt = sql.NullTime{Time: time.Now(), Valid: true}

	if jobErr != nil {
		h.logger.Errorw("job failed", "error", jobErr)

		if job.Retries < job.MaxRetries {
			job.Status = model.QueueJobStatusRetry
			job.RunAfter = queue.CalculateBackoff(job.Retries)
		} else {
			job.Status = model.QueueJobStatusFailed
		}

		job.Error = model.NewNullString(jobErr.Error())
	} else {
		job.Status = model.QueueJobStatusProcessed
		processed = true
	}

	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), resultWriteTimeout)
	defer cancel()

	// only write the outcome if we still hold the lease, so that a job that overran its lease
	// doesn't clobber the result of another attempt that has since claimed it
	result, updateErr := h.query.QueueJob.WithContext(writeCtx).Where(
		h.query.QueueJob.ID.Eq(job.ID),
		h.query.QueueJob.RunAfter.Eq(leaseUntil),
	).Updates(job)

	switch {
	case updateErr != nil:
		err = updateErr
		processed = false

		h.logger.Errorw("error recording job outcome", "job_id", jobID, "error", err)
	case result.RowsAffected == 0:
		processed = false

		h.logger.Warnw("lost lease on job before recording its outcome", "job_id", jobID)
	case processed:
		h.logger.Debugw("job processed", "job_id", jobID)
	}

	return jobID, processed, err
}

// claimJob finds the next runnable job and takes a lease on it, returning nil if no job is runnable.
// The returned job is as it was before the lease was taken; leaseUntil is the run_after value of the lease.
func (h *serverHandler) claimJob(
	ctx context.Context,
	conds ...gen.Condition,
) (job *model.QueueJob, leaseUntil time.Time, err error) {
	// Postgres stores microseconds; truncate so the lease can be matched exactly when recording the outcome.
	leaseUntil = time.Now().Add(h.JobTimeout + leaseGrace).Truncate(time.Microsecond)

	err = h.query.Transaction(func(tx *dao.Query) error {
		found, findErr := tx.QueueJob.WithContext(ctx).Where(
			append(
				conds,
				tx.QueueJob.Queue.Eq(h.Queue),
				tx.QueueJob.Status.In(
					string(model.QueueJobStatusPending),
					string(model.QueueJobStatusRetry),
				),
				tx.QueueJob.RunAfter.Lte(time.Now()),
			)...,
		).Order(
			tx.QueueJob.Status.Eq(string(model.QueueJobStatusRetry)),
			tx.QueueJob.Priority,
			tx.QueueJob.RunAfter,
		).Clauses(clause.Locking{
			Strength: "UPDATE",
			Options:  "SKIP LOCKED",
		}).First()
		if findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return nil
			}

			return findErr
		}

		if _, leaseErr := tx.QueueJob.WithContext(ctx).Where(
			tx.QueueJob.ID.Eq(found.ID),
		).UpdateSimple(tx.QueueJob.RunAfter.Value(leaseUntil)); leaseErr != nil {
			return leaseErr
		}

		job = found

		return nil
	})

	return job, leaseUntil, err
}

var ErrJobExceededDeadline = errors.New("the job did not complete before its deadline")
