package debug

import "sync"

type RequestRecord struct {
	Operation  string `json:"operation"`
	URL        string `json:"url"`
	StatusCode int    `json:"status_code"`
	DurationMS int64  `json:"duration_ms"`
	ItemsCount int    `json:"items_count"`
	Error      string `json:"error"`
}

type Recorder struct {
	mu      sync.Mutex
	limit   int
	records []RequestRecord
}

func NewRecorder(limit int) *Recorder {
	if limit <= 0 {
		limit = 100
	}
	return &Recorder{limit: limit}
}

func (r *Recorder) Record(record RequestRecord) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, record)
	if len(r.records) > r.limit {
		r.records = append([]RequestRecord(nil), r.records[len(r.records)-r.limit:]...)
	}
}

func (r *Recorder) List() []RequestRecord {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]RequestRecord, len(r.records))
	copy(out, r.records)
	return out
}
