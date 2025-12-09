package pihole

import (
	"encoding/json"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

// getLastQueryEntry reads the LAST_QUERY_DATA env variable and parses it as QueryHistoryEntry, or returns zero value if not set or invalid.
func getLastQueryEntry() QueryHistoryEntry {
	var lastQuery QueryHistoryEntry

	// run one minute back in time
	now := time.Now().Unix() - 60
	var defaultLastQuery = QueryHistoryEntry{
		Timestamp: now - 60,
		Total:     0,
		Cached:    0,
		Blocked:   0,
		Forwarded: 0,
	}

	lastQueryStr := os.Getenv("LAST_QUERY_DATA")
	if lastQueryStr != "" {
		if err := json.Unmarshal([]byte(lastQueryStr), &lastQuery); err != nil {
			log.Infof("Failed to parse LAST_QUERY_DATA: %v", err)
			setLastQueryEntry(defaultLastQuery)
			return defaultLastQuery
		}
		return lastQuery
	}
	setLastQueryEntry(defaultLastQuery)
	return defaultLastQuery
}

func setLastQueryEntry(entry QueryHistoryEntry) {
	entryBytes, err := json.Marshal(entry)
	if err != nil {
		log.Infof("Failed to marshal last query entry: %v", err)
		return
	}
	if err := os.Setenv("LAST_QUERY_DATA", string(entryBytes)); err != nil {
		log.Infof("Failed to set LAST_QUERY_DATA env variable: %v", err)
	}

	log.Infof("Last Query Timestamp %d", entry.Timestamp)
}
