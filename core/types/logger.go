package types

import (
	"encoding/json"
	"fmt"
	"time"
)

type LogEntry struct {
	Timestamp string `json:"time"`
	Level     string `json:"level"`
	Module    string `json:"module"`
	Message   string `json:"msg"`
	NodeID    string `json:"node_id,omitempty"`
}

func Log(level, module, msg, nodeID string) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Module:    module,
		Message:   msg,
		NodeID:    nodeID,
	}
	data, _ := json.Marshal(entry)
	fmt.Println(string(data))
}
