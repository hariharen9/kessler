package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// ScanHistoryEntry records the result of a single scan.
type ScanHistoryEntry struct {
	Timestamp    time.Time `json:"timestamp"`
	ScanPath     string    `json:"scan_path"`
	ProjectCount int       `json:"project_count"`
	TotalSize    int64     `json:"total_size"`
	FreedSpace   int64     `json:"freed_space,omitempty"`
}

// ScanHistory holds recent scan entries.
type ScanHistory struct {
	Entries []ScanHistoryEntry `json:"entries"`
}

func historyPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.TempDir()
	}
	return filepath.Join(configDir, "kessler", "history.json")
}

// LoadHistory reads scan history from disk.
func LoadHistory() ScanHistory {
	data, err := os.ReadFile(historyPath())
	if err != nil {
		return ScanHistory{}
	}
	var h ScanHistory
	if err := json.Unmarshal(data, &h); err != nil {
		return ScanHistory{}
	}
	return h
}

// SaveEntry appends a scan entry and keeps the last 20.
func SaveEntry(entry ScanHistoryEntry) {
	h := LoadHistory()
	h.Entries = append(h.Entries, entry)

	// Keep only the last 20 entries
	if len(h.Entries) > 20 {
		h.Entries = h.Entries[len(h.Entries)-20:]
	}

	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return
	}

	path := historyPath()
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, data, 0o644)
}

// LastEntry returns the most recent scan entry, or nil if none.
func (h ScanHistory) LastEntry() *ScanHistoryEntry {
	if len(h.Entries) == 0 {
		return nil
	}
	return &h.Entries[len(h.Entries)-1]
}
