package devastator

import (
	"log"
	"sync"

	"github.com/nbusy/cmap"
	"github.com/nbusy/neptulon/jsonrpc"
)

// Queue is a message queue for queueing and sending messages to users.
type Queue struct {
	conns   *cmap.CMap      // user ID -> conn ID
	router  *jsonrpc.Router // route to send and receive messages through
	reqs    *cmap.CMap      // user ID -> []queuedRequest
	mutexes *cmap.CMap      // user ID -> sync.RWMutex
}

// NewQueue creates a new queue object.
func NewQueue() Queue {
	q := Queue{
		conns:   cmap.New(),
		reqs:    cmap.New(),
		mutexes: cmap.New(),
	}

	return q
}

// Middleware registers a queue middleware to register user/connection IDs
// for connecting users (upon their first incoming-message).
func (q *Queue) Middleware(s *jsonrpc.Server) {
	s.ReqMiddleware(func(ctx *jsonrpc.ReqCtx) {
		q.SetConn(ctx.Conn.Data.Get("userid").(string), ctx.Conn.ID)
	})
	s.ResMiddleware(func(ctx *jsonrpc.ResCtx) {
		q.SetConn(ctx.Conn.Data.Get("userid").(string), ctx.Conn.ID)
	})
	s.NotMiddleware(func(ctx *jsonrpc.NotCtx) {
		q.SetConn(ctx.Conn.Data.Get("userid").(string), ctx.Conn.ID)
	})
}

// SetRouter registers the JSON-RPC router to be used for sending queued messages and expecting responses through.
func (q *Queue) SetRouter(r *jsonrpc.Router) {
	q.router = r
}

// SetConn associates a user with a connection by ID.
// If there are pending messages for the user, they are started to be send immediately.
func (q *Queue) SetConn(userID, connID string) {
	if _, ok := q.conns.GetOk(userID); !ok {
		q.conns.Set(userID, connID)
		q.mutexes.Set(userID, sync.RWMutex{})
		go q.processQueue(userID)
	}
}

// RemoveConn removes a user's associated connection ID.
func (q *Queue) RemoveConn(userID string) {
	q.conns.Delete(userID)
	q.mutexes.Delete(userID)
}

// AddRequest queues a request message to be sent to the given user.
func (q *Queue) AddRequest(userID string, method string, params interface{}, resHandler func(ctx *jsonrpc.ResCtx)) error {
	r := queuedRequest{Method: method, Params: params, ResHandler: resHandler}

	go func() {
		if rs, ok := q.reqs.GetOk(userID); ok {
			q.reqs.Set(userID, append(rs.([]queuedRequest), r))
		} else {
			q.reqs.Set(userID, []queuedRequest{{Method: method, Params: params, ResHandler: resHandler}})
		}

		go q.processQueue(userID)
	}()

	return nil
}

type queuedRequest struct {
	Method     string
	Params     interface{}
	ResHandler func(ctx *jsonrpc.ResCtx)
}

// todo: prevent concurrent runs of processQueue or make []queuedRequest thread-safe
func (q *Queue) processQueue(userID string) {
	connID, ok := q.conns.GetOk(userID)
	if !ok {
		return
	}

	mutex := q.mutexes.Get(userID).(sync.RWMutex)
	mutex.Lock()
	if ireqs, ok := q.reqs.GetOk(userID); ok {
		reqs := ireqs.([]queuedRequest)
		for i, req := range reqs {
			if err := q.router.SendRequest(connID.(string), req.Method, req.Params, req.ResHandler); err != nil {
				log.Fatal(err) // todo: log fatal only in debug mode
			} else {
				reqs, reqs[len(reqs)-1] = append(reqs[:i], reqs[i+1:]...), queuedRequest{} // todo: this might not be needed if function is not a pointer val
			}
		}

		q.reqs.Set(userID, reqs)
	}
	mutex.Unlock()
}
