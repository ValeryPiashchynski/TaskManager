package rmq

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/nats-io/go-nats"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type Server struct {
	e      endpoint.Endpoint
	dec    DecodeRequestFunc
	enc    EncodeResponseFunc
	before []ServerRequestFunc
	after  []ServerResponseFunc
	logger logrus.Logger

	FlushInterval time.Duration
	Conn          *amqp.Channel
	MaxBatchSize  int
	WorkerCount   int
	QueueSize     int
	QueueName     string
	// StopCh        chan os.Signal
	// QueueCh       chan *nats.Msg
	// ErrorCh       chan error
}

func NewServer(
	e endpoint.Endpoint,
	dec DecodeRequestFunc,
	enc EncodeResponseFunc,

	RqChannel *amqp.Channel,
	Logger logrus.Logger,
	// StopCh chan os.Signal,
	// MaxBatchSize int,
	// WorkerCount int,
	// QueueSize int,
	// QueueName string,
	// FlushInterval time.Duration,
	// ErrorCh chan error,

	options ...ServerOption,

) *Server {
	s := &Server{
		e:      e,
		dec:    dec,
		enc:    enc,
		logger: Logger,

		Conn: RqChannel,
		// FlushInterval: FlushInterval, //FlushInterval
		// MaxBatchSize:  MaxBatchSize,
		// WorkerCount:   WorkerCount,
		// QueueSize:     QueueSize,
		// QueueName:     QueueName,
		// StopCh:        StopCh,
		// QueueCh:       make(chan *nats.Msg), //QueueCh
		// ErrorCh:       ErrorCh,
	}

	for _, option := range options {
		option(s)
	}

	go s.StartNatsWorkers(StopCh, NatsConn, QueueName)
	return s
}

// ServerOption sets an optional parameter for servers.
type ServerOption func(*Server)

func ServerBefore(before ...ServerRequestFunc) ServerOption {
	return func(s *Server) {
		s.before = append(s.before, before...)
	}
}

func ServerAfter(after ...ServerResponseFunc) ServerOption {
	return func(s *Server) {
		s.after = append(s.after, after...)
	}
}

func ServerErrorLogger(logger logrus.Logger) ServerOption {
	return func(s *Server) { s.logger = logger }
}

func (s *Server) MsgHandler(msg *nats.Msg) {
	s.QueueCh <- msg
	logrus.WithFields(
		logrus.Fields{
			"added": msg.Subject,
		},
	).Info("MsgHandler")
}

func (s *Server) worker(id int) {
	var buffer []*nats.Msg
	logrus.WithFields(
		logrus.Fields{
			"worker started, N: ": id,
		},
	).Info("MsgHandler")

	timer := time.NewTimer(s.FlushInterval)
	defer timer.Stop()

	for {
		select {
		case payload, opened := <-s.QueueCh:
			if !opened {
				if len(buffer) == 0 {
					logrus.WithFields(
						logrus.Fields{
							"buffer is empty, get stop signal, worker N: ": id,
						},
					).Info("worker")

					return
				}
				s.flush(id, buffer, "get stop signal")
				return
			}

			buffer = append(buffer, payload)
			if len(buffer) >= s.MaxBatchSize {
				s.flush(id, buffer, "max size reached")
				buffer = nil
				timer.Reset(s.FlushInterval)
			}
		case <-timer.C:
			//To prevent flushing empty buffer
			if len(buffer) == 0 {
				buffer = nil
				timer.Reset(s.FlushInterval)
			} else {
				s.flush(id, buffer, "time limit reached")
				buffer = nil
				timer.Reset(s.FlushInterval)
			}
		}
	}
}

func (s *Server) flush(workerId int, buffer []*nats.Msg, reason string) {
	defer func(tt time.Time) {
		logrus.WithFields(
			logrus.Fields{
				"worker N ": workerId,
				"flushed":   len(buffer),
				"reason":    reason,
				"ts":        time.Since(tt),
			},
		).Info("flush")
	}(time.Now())

	for _, m := range buffer {
		ctx := context.TODO()
		request, err := s.dec(ctx, m)
		if err != nil {
			s.logger.Error("err", err)
			s.ErrorCh <- err
			return
		}

		response, err := s.e(ctx, request)
		if err != nil {
			s.logger.Error("err", err)
			s.ErrorCh <- err
			return
		}

		payload, err := s.enc(ctx, response)
		if err != nil {
			s.logger.Error("err", err)
			s.ErrorCh <- err
			return
		}

		s.Conn.Publish(m.Reply, payload)
	}
}

func (s *Server) StartNatsWorkers(stop chan os.Signal, nc *nats.Conn, queueName string) {
	logrus.WithFields(
		logrus.Fields{
			"start queue": queueName,
		},
	).Info("Start workers")

	defer logrus.WithFields(
		logrus.Fields{
			"end queue": queueName,
		},
	).Info("workers stopped")

	s.ErrorCh = make(chan error)
	s.QueueCh = make(chan *nats.Msg, s.QueueSize)
	//defer close(s.ErrorCh)
	//defer close(s.QueueCh)

	wg := sync.WaitGroup{}
	wg.Add(s.WorkerCount)
	for i := 0; i < s.WorkerCount; i++ {
		go func(id int) {
			defer wg.Done()
			s.worker(id)
		}(i)
	}

	<-stop
	nc.Close()
	close(s.QueueCh)
	wg.Wait()
}
