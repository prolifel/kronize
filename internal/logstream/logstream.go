package logstream

import (
	"database/sql"
	"sync"

	"kronize/internal/db"
)

type Event struct {
	Type string `json:"type"`
	Seq  int64  `json:"seq"`
	Data string `json:"data,omitempty"`
}

type Snapshot struct {
	Status     string `json:"status"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   *int   `json:"exit_code"`
	DurationMs *int64 `json:"duration_ms"`
}

type execState struct {
	mu   sync.Mutex
	seq  int64
	subs map[chan Event]struct{}
}

type Hub struct {
	mu    sync.Mutex
	db    *sql.DB
	execs map[int64]*execState
}

func New(database *sql.DB) *Hub {
	return &Hub{
		db:    database,
		execs: make(map[int64]*execState),
	}
}

func (h *Hub) state(execID int64) *execState {
	h.mu.Lock()
	defer h.mu.Unlock()
	st := h.execs[execID]
	if st == nil {
		st = &execState{subs: make(map[chan Event]struct{})}
		h.execs[execID] = st
	}
	return st
}

func (h *Hub) Append(execID int64, stream, chunk string) error {
	if chunk == "" {
		return nil
	}
	st := h.state(execID)
	st.mu.Lock()
	defer st.mu.Unlock()

	if err := db.AppendExecutionOutput(h.db, execID, stream, chunk); err != nil {
		return err
	}
	st.seq++
	h.publishLocked(st, Event{Type: stream, Seq: st.seq, Data: chunk})
	return nil
}

func (h *Hub) PublishStatus(execID int64, data string) {
	st := h.state(execID)
	st.mu.Lock()
	defer st.mu.Unlock()
	st.seq++
	h.publishLocked(st, Event{Type: "status", Seq: st.seq, Data: data})
}

func (h *Hub) SubscribeWithSnapshot(execID int64, snapshot func() (Snapshot, error)) (<-chan Event, Snapshot, func(), error) {
	st := h.state(execID)
	st.mu.Lock()
	defer st.mu.Unlock()

	snap, err := snapshot()
	if err != nil {
		return nil, Snapshot{}, nil, err
	}

	ch := make(chan Event, 64)
	st.subs[ch] = struct{}{}

	cancel := func() {
		st.mu.Lock()
		defer st.mu.Unlock()
		if _, ok := st.subs[ch]; ok {
			delete(st.subs, ch)
			close(ch)
		}
	}

	return ch, snap, cancel, nil
}

func (h *Hub) publishLocked(st *execState, ev Event) {
	for ch := range st.subs {
		select {
		case ch <- ev:
		default:
		}
	}
}
