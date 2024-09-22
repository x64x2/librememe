package downloader

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/sirupsen/logrus"

)

var SharedWorker *DownloadWorker

type DownloadWorker struct {
	msgChan chan *DownloadOp
	cancel  context.CancelFunc
	running *atomic.Bool
	wg      *sync.WaitGroup
	client  *http.Client
}

func NewDownloadWorker() *DownloadWorker {
	return &DownloadWorker{
		running: &atomic.Bool{},
		client: &http.Client{
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 1 * time.Minute,
				}).Dial,
				TLSHandshakeTimeout:   30 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
	}
}

func (w *DownloadWorker) Add(op *DownloadOp) error {
	if !w.running.Load() {
		return errors.New("this worker is not running")
	}

	w.msgChan <- op

	return nil
}

func (w *DownloadWorker) Start(numWorkers int, buffer int) error {
	if !w.running.CompareAndSwap(false, true) {
		return errors.New("this worker is already running")
	}

	w.msgChan = make(chan *DownloadOp, buffer)

	var ctx context.Context
	ctx, w.cancel = context.WithCancel(context.Background())
	w.wg = &sync.WaitGroup{}
	for i := 0; i < numWorkers; i++ {
		w.wg.Add(1)
		go w.worker(ctx, i)
	}

	return nil
}

func (w *DownloadWorker) Wait() {
	if w.running.Load() && w.wg != nil {
		w.wg.Wait()
	}
}

func (w *DownloadWorker) Done() {
	if w.running.Load() && w.msgChan != nil {
		close(w.msgChan)
	}
}

func (w *DownloadWorker) Stop() error {
	if !w.running.Load() {
		return errors.New("this worker is not running")
	}

	w.cancel()
	w.cancel = nil

	w.wg.Wait()
	w.msgChan = nil

	w.running.Store(false)

	return nil
}

func (w *DownloadWorker) worker(ctx context.Context, i int) {
	defer func() {
		log.Debugf("Worker %d done", i)
		w.wg.Done()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case op, ok := <-w.msgChan:
			if !ok || op == nil || ctx.Err() != nil {
				return
			}
			w.downloadRequest(ctx, op)
		}
	}
}

func (w *DownloadWorker) downloadRequest(ctx context.Context, op *DownloadOp) {
	log.Debugf("Started downloading file '%s'", op.Destination)

	req := op.Request.WithContext(ctx)

	bo := backoff.WithContext(
		backoff.NewExponentialBackOff(backoff.WithInitialInterval(5*time.Second)),
		ctx,
	)
	hash, err := backoff.RetryNotifyWithData(func() ([]byte, error) {
		start := time.Now()
		res, err := w.client.Do(req)
		if err != nil {
			if err == context.Canceled || err == context.DeadlineExceeded {
				return nil, backoff.Permanent(err)
			}
			return nil, fmt.Errorf("error requesting file: %w", err)
		}

		defer func() {
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
		}()

		if res.StatusCode >= 400 {
			err = fmt.Errorf("invalid response status code: %d", res.StatusCode)
			switch res.StatusCode {
			case http.StatusUnauthorized, http.StatusForbidden, http.StatusBadRequest:
				return nil, backoff.Permanent(err)
			default:
				return nil, err
			}
		}

		h := sha256.New()
		tr := io.TeeReader(res.Body, h)

		err = fs.SharedFS.Put(ctx, op.Destination, tr)
		if err != nil {
			return nil, fmt.Errorf("failed to download file '%s': %w", op.Destination, err)
		}

		dur := time.Now().Sub(start)
		log.Infof("⬇️ Downloaded file '%s' in %.2fs", op.Destination, dur.Seconds())

		return h.Sum(nil), nil
	}, bo, func(err error, d time.Duration) {
		log.Warnf("Failed to download file '%s' with error: '%v'. Retrying in %v", op.Destination, err, d)
	})
	if err != nil {
		log.Errorf("Permanent failure while downloading file '%s'. Error: '%v'", op.Destination, err)
		return
	}

	if op.OnComplete != nil {
		err = op.OnComplete(hash)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			log.Panicf("Failed to execute OnComplete method for file '%s': '%v'", op.Destination, err)
			return
		}
	}
	log.Debugf("All done for file '%s'", op.Destination)
}
