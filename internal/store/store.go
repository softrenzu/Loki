package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// LogEntry is RooomLog's canonical log record. Timestamp is Unix nanoseconds.
type LogEntry struct {
	Timestamp int64             `json:"timestamp"`
	Tenant    string            `json:"tenant,omitempty"`
	Message   string            `json:"message"`
	Labels    map[string]string `json:"labels,omitempty"`
	Fields    map[string]any    `json:"fields,omitempty"`
	TraceID   string            `json:"trace_id,omitempty"`
	SpanID    string            `json:"span_id,omitempty"`
}

type Filter struct {
	Tenant string
	Query  string
	Labels map[string]string
	From   int64
	To     int64
	Limit  int
}

type Stats struct {
	Entries    int `json:"entries"`
	Tokens     int `json:"tokens"`
	LabelPairs int `json:"label_pairs"`
}

type Store struct {
	mu      sync.RWMutex
	entries []LogEntry
	byToken map[string][]int
	byLabel map[string][]int
	wal     *os.File
	walPath string
	subs    map[chan LogEntry]struct{}
}

var tokenRE = regexp.MustCompile(`[\p{L}\p{N}_./:@-]+`)

func Open(dir string) (*Store, error) {
	if dir == "" {
		return nil, errors.New("data dir is required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	p := filepath.Join(dir, "rooomlog.jsonl")
	s := &Store{byToken: map[string][]int{}, byLabel: map[string][]int{}, walPath: p, subs: map[chan LogEntry]struct{}{}}
	if f, err := os.Open(p); err == nil {
		sc := bufio.NewScanner(f)
		buf := make([]byte, 64*1024)
		sc.Buffer(buf, 16*1024*1024)
		for sc.Scan() {
			var e LogEntry
			if json.Unmarshal(sc.Bytes(), &e) == nil {
				s.addToIndex(e)
			}
		}
		_ = f.Close()
		if err := sc.Err(); err != nil {
			return nil, fmt.Errorf("read wal: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	s.wal = f
	return s, nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.wal != nil {
		return s.wal.Close()
	}
	return nil
}

func normalize(e *LogEntry) {
	if e.Timestamp == 0 {
		e.Timestamp = time.Now().UnixNano()
	}
	if e.Tenant == "" {
		e.Tenant = "default"
	}
	if e.Labels == nil {
		e.Labels = map[string]string{}
	}
	if e.Fields == nil {
		e.Fields = map[string]any{}
	}
}

func (s *Store) Append(e LogEntry) error {
	normalize(&e)
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if s.wal == nil {
		s.mu.Unlock()
		return errors.New("store closed")
	}
	if _, err = s.wal.Write(append(b, '\n')); err != nil {
		s.mu.Unlock()
		return err
	}
	if err = s.wal.Sync(); err != nil {
		s.mu.Unlock()
		return err
	}
	s.addToIndexLocked(e)
	subscribers := make([]chan LogEntry, 0, len(s.subs))
	for ch := range s.subs {
		subscribers = append(subscribers, ch)
	}
	s.mu.Unlock()
	for _, ch := range subscribers {
		select {
		case ch <- e:
		default:
		}
	}
	return nil
}

func (s *Store) addToIndex(e LogEntry) { normalize(&e); s.addToIndexLocked(e) }
func (s *Store) addToIndexLocked(e LogEntry) {
	idx := len(s.entries)
	s.entries = append(s.entries, e)
	seen := map[string]struct{}{}
	addTok := func(v string) {
		for _, t := range tokenRE.FindAllString(strings.ToLower(v), -1) {
			if len(t) < 2 {
				continue
			}
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			s.byToken[t] = append(s.byToken[t], idx)
		}
	}
	addTok(e.Message)
	addTok(e.TraceID)
	addTok(e.SpanID)
	for k, v := range e.Labels {
		s.byLabel[labelKey(e.Tenant, k, v)] = append(s.byLabel[labelKey(e.Tenant, k, v)], idx)
		addTok(k)
		addTok(v)
	}
	for k, v := range e.Fields {
		addTok(k)
		addTok(fmt.Sprint(v))
	}
}

func labelKey(tenant, k, v string) string { return tenant + "\x00" + k + "\x00" + v }

func (s *Store) Query(f Filter) []LogEntry {
	if f.Tenant == "" {
		f.Tenant = "default"
	}
	if f.Limit <= 0 || f.Limit > 10000 {
		f.Limit = 200
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	var candidate []int
	choose := func(list []int) {
		if candidate == nil || len(list) < len(candidate) {
			candidate = list
		}
	}
	for k, v := range f.Labels {
		choose(s.byLabel[labelKey(f.Tenant, k, v)])
	}
	for _, tok := range tokenRE.FindAllString(strings.ToLower(f.Query), -1) {
		if len(tok) >= 2 {
			choose(s.byToken[tok])
		}
	}
	if candidate == nil {
		candidate = make([]int, len(s.entries))
		for i := range s.entries {
			candidate[i] = i
		}
	}
	q := strings.ToLower(strings.TrimSpace(f.Query))
	out := make([]LogEntry, 0, min(f.Limit, len(candidate)))
	for i := len(candidate) - 1; i >= 0 && len(out) < f.Limit; i-- {
		e := s.entries[candidate[i]]
		if e.Tenant != f.Tenant {
			continue
		}
		if f.From != 0 && e.Timestamp < f.From {
			continue
		}
		if f.To != 0 && e.Timestamp > f.To {
			continue
		}
		ok := true
		for k, v := range f.Labels {
			if e.Labels[k] != v {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		if q != "" && !entryContains(e, q) {
			continue
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp > out[j].Timestamp })
	return out
}

func entryContains(e LogEntry, q string) bool {
	if strings.Contains(strings.ToLower(e.Message), q) || strings.Contains(strings.ToLower(e.TraceID), q) || strings.Contains(strings.ToLower(e.SpanID), q) {
		return true
	}
	for k, v := range e.Labels {
		if strings.Contains(strings.ToLower(k+"="+v), q) {
			return true
		}
	}
	for k, v := range e.Fields {
		if strings.Contains(strings.ToLower(k+"="+fmt.Sprint(v)), q) {
			return true
		}
	}
	// Multi-token queries should still match when tokens occur in different structured fields.
	toks := tokenRE.FindAllString(q, -1)
	if len(toks) > 1 {
		blob, _ := json.Marshal(e)
		low := strings.ToLower(string(blob))
		for _, t := range toks {
			if !strings.Contains(low, t) {
				return false
			}
		}
		return true
	}
	return false
}

func (s *Store) Subscribe() (<-chan LogEntry, func()) {
	ch := make(chan LogEntry, 128)
	s.mu.Lock()
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	cancel := func() {
		s.mu.Lock()
		if _, ok := s.subs[ch]; ok {
			delete(s.subs, ch)
			close(ch)
		}
		s.mu.Unlock()
	}
	return ch, cancel
}

func (s *Store) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Stats{Entries: len(s.entries), Tokens: len(s.byToken), LabelPairs: len(s.byLabel)}
}

func (s *Store) DeleteBefore(cutoff int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := make([]LogEntry, 0, len(s.entries))
	deleted := 0
	for _, e := range s.entries {
		if e.Timestamp < cutoff {
			deleted++
		} else {
			kept = append(kept, e)
		}
	}
	if deleted == 0 {
		return 0, nil
	}
	tmp := s.walPath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return 0, err
	}
	bw := bufio.NewWriter(f)
	for _, e := range kept {
		b, _ := json.Marshal(e)
		if _, err = bw.Write(append(b, '\n')); err != nil {
			_ = f.Close()
			return 0, err
		}
	}
	if err = bw.Flush(); err != nil {
		_ = f.Close()
		return 0, err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return 0, err
	}
	if err = f.Close(); err != nil {
		return 0, err
	}
	if s.wal != nil {
		_ = s.wal.Close()
	}
	if err = os.Rename(tmp, s.walPath); err != nil {
		return 0, err
	}
	s.wal, err = os.OpenFile(s.walPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	s.entries = kept
	s.byToken = map[string][]int{}
	s.byLabel = map[string][]int{}
	entriesCopy := append([]LogEntry(nil), kept...)
	s.entries = nil
	for _, e := range entriesCopy {
		s.addToIndexLocked(e)
	}
	return deleted, nil
}
