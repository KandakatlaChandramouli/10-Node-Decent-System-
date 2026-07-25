package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ExperimentMetrics struct {
	ExperimentName  string    `json:"experiment_name"`
	Timestamp       time.Time `json:"timestamp"`
	TotalNodes      int       `json:"total_nodes"`
	TargetTPS       int       `json:"target_tps"`
	AchievedTPS     int       `json:"achieved_tps"`
	AvgLatencyMs    int       `json:"avg_latency_ms"`
	MemoryFootprint string    `json:"memory_footprint"`
	Consensus       string    `json:"consensus"`
	Status          string    `json:"status"`
}

type MetricExporter struct {
	Metrics ExperimentMetrics
}

func NewMetricExporter(m ExperimentMetrics) *MetricExporter {
	return &MetricExporter{Metrics: m}
}

func (e *MetricExporter) ExportJSON(outPath string) error {
	_ = os.MkdirAll(filepath.Dir(outPath), 0755)
	data, err := json.MarshalIndent(e.Metrics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, data, 0644)
}

func (e *MetricExporter) ExportCSV(outPath string) error {
	_ = os.MkdirAll(filepath.Dir(outPath), 0755)
	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"ExperimentName", "Timestamp", "TotalNodes", "TargetTPS", "AchievedTPS", "AvgLatencyMs", "MemoryFootprint", "Consensus", "Status"}
	if err := writer.Write(headers); err != nil {
		return err
	}

	row := []string{
		e.Metrics.ExperimentName,
		e.Metrics.Timestamp.Format(time.RFC3339),
		fmt.Sprintf("%d", e.Metrics.TotalNodes),
		fmt.Sprintf("%d", e.Metrics.TargetTPS),
		fmt.Sprintf("%d", e.Metrics.AchievedTPS),
		fmt.Sprintf("%d", e.Metrics.AvgLatencyMs),
		e.Metrics.MemoryFootprint,
		e.Metrics.Consensus,
		e.Metrics.Status,
	}
	return writer.Write(row)
}

func (e *MetricExporter) ExportMarkdown(outPath string) error {
	_ = os.MkdirAll(filepath.Dir(outPath), 0755)
	md := fmt.Sprintf(`# Experiment Results: %s

- **Timestamp:** %s
- **Status:** %s

| Parameter | Value |
| :--- | :--- |
| **Total Nodes** | %d |
| **Target TPS** | %d TPS |
| **Achieved TPS** | %d TPS |
| **Avg Latency** | %d ms |
| **Memory Footprint** | %s |
| **Consensus Engine** | %s |
`,
		e.Metrics.ExperimentName,
		e.Metrics.Timestamp.Format(time.RFC3339),
		e.Metrics.Status,
		e.Metrics.TotalNodes,
		e.Metrics.TargetTPS,
		e.Metrics.AchievedTPS,
		e.Metrics.AvgLatencyMs,
		e.Metrics.MemoryFootprint,
		e.Metrics.Consensus,
	)
	return os.WriteFile(outPath, []byte(md), 0644)
}
