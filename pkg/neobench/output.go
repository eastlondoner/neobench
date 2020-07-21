package neobench

import (
	"fmt"
	"github.com/codahale/hdrhistogram"
	"io"
	"os"
	"strings"
	"time"
)

type ProgressReport struct {
	Section      string
	Step         string
	Completeness float64
}

type ThroughputResult struct {
	Scenario           string
	TotalRatePerSecond float64
}

type LatencyResult struct {
	Scenario       string
	TotalHistogram *hdrhistogram.Histogram
}

type Output interface {
	ReportProgress(report ProgressReport)
	ReportThroughputResult(result ThroughputResult)
	ReportLatencyResult(result LatencyResult)
	Errorf(format string, a ...interface{})
}

func NewOutput(name string) (Output, error) {

	if name == "auto" {
		fi, _ := os.Stdout.Stat()
		if fi.Mode() & os.ModeCharDevice == 0 {
			return &CsvOutput{
				ErrStream:          os.Stderr,
				OutStream:          os.Stdout,
			}, nil
		} else {
			return &InteractiveOutput{
				ErrStream:          os.Stderr,
				OutStream:          os.Stdout,
			}, nil
		}
	}
	if name == "interactive" {
		return &InteractiveOutput{
			ErrStream:          os.Stderr,
			OutStream:          os.Stdout,
		}, nil
	}
	if name == "csv" {
		return &CsvOutput{
			ErrStream:          os.Stderr,
			OutStream:          os.Stdout,
		}, nil
	}
	return nil, fmt.Errorf("unknown output format: %s, supported formats are 'auto', 'interactive' and 'csv'", name)
}

type InteractiveOutput struct {
	ErrStream io.Writer
	OutStream io.Writer
	// Used to rate-limit progress reporting
	LastProgressReport ProgressReport
	LastProgressTime   time.Time
}

func (o *InteractiveOutput) ReportProgress(report ProgressReport) {
	now := time.Now()
	if report.Section == o.LastProgressReport.Section && report.Step == o.LastProgressReport.Step && now.Sub(o.LastProgressTime).Seconds() < 10 {
		return
	}
	o.LastProgressReport = report
	o.LastProgressTime = now
	_, err := fmt.Fprintf(o.ErrStream, "[%s][%s] %.02f%%\n", report.Section, report.Step, report.Completeness*100)
	if err != nil {
		panic(err)
	}
}

func (o *InteractiveOutput) ReportThroughputResult(result ThroughputResult) {
	s := strings.Builder{}

	s.WriteString("== Benchmark Completed! ==\n")
	s.WriteString(fmt.Sprintf("Scenario: %s\n", result.Scenario))
	s.WriteString(fmt.Sprintf("Rate: %.03f transactions per second\n", result.TotalRatePerSecond))

	_, err := fmt.Fprintf(o.OutStream, s.String())
	if err != nil {
		panic(err)
	}
}

func (o *InteractiveOutput) ReportLatencyResult(result LatencyResult) {
	histo := result.TotalHistogram

	s := strings.Builder{}

	s.WriteString("== Benchmark Completed! ==\n")
	s.WriteString(fmt.Sprintf("Scenario: %s\n", result.Scenario))
	s.WriteString(fmt.Sprintf("Total Transactions: %d\n", histo.TotalCount()))
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("Latency summary:\n"))
	s.WriteString(fmt.Sprintf("  Min:    %.3fms\n", float64(histo.Min()) / 1000.0))
	s.WriteString(fmt.Sprintf("  Mean:   %.3fms\n", histo.Mean() / 1000.0))
	s.WriteString(fmt.Sprintf("  Max:    %.3fms\n", float64(histo.Max()) / 1000.0))
	s.WriteString(fmt.Sprintf("  Stddev: %.3fms\n", histo.StdDev() / 1000.0))
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("Latency distribution:\n"))
	s.WriteString(fmt.Sprintf("  P50.000: %.03fms\n", float64(histo.ValueAtQuantile(50)) / 1000.0))
	s.WriteString(fmt.Sprintf("  P75.000: %.03fms\n", float64(histo.ValueAtQuantile(75)) / 1000.0))
	s.WriteString(fmt.Sprintf("  P95.000: %.03fms\n", float64(histo.ValueAtQuantile(95)) / 1000.0))
	s.WriteString(fmt.Sprintf("  P99.000: %.03fms\n", float64(histo.ValueAtQuantile(99)) / 1000.0))
	s.WriteString(fmt.Sprintf("  P99.999: %.03fms\n", float64(histo.ValueAtQuantile(99.999)) / 1000.0))

	_, err := fmt.Fprint(o.OutStream, s.String())
	if err != nil {
		panic(err)
	}

}

func (o *InteractiveOutput) Errorf(format string, a ...interface{}) {
	_, err := fmt.Fprintf(o.ErrStream, "ERROR: %s\n", fmt.Sprintf(format, a...))
	if err != nil {
		panic(err)
	}
}

// Writes simple progress to stderr, and then a result for easy import into eg. a spreadsheet or other app
// in CSV format to stdout
type CsvOutput struct {
	ErrStream io.Writer
	OutStream io.Writer
	// Used to rate-limit progress reporting
	LastProgressReport ProgressReport
	LastProgressTime   time.Time
}

func (o *CsvOutput) ReportProgress(report ProgressReport) {
	now := time.Now()
	if report.Section == o.LastProgressReport.Section && report.Step == o.LastProgressReport.Step && now.Sub(o.LastProgressTime).Seconds() < 10 {
		return
	}
	o.LastProgressReport = report
	o.LastProgressTime = now
	_, err := fmt.Fprintf(o.ErrStream, "[%s][%s] %.02f%%\n", report.Section, report.Step, report.Completeness*100)
	if err != nil {
		panic(err)
	}
}

func (o *CsvOutput) ReportThroughputResult(result ThroughputResult) {
	_, err := fmt.Fprintf(o.OutStream, "scenario,transactions_per_second\n\"%s\",%.03f\n", result.Scenario, result.TotalRatePerSecond)
	if err != nil {
		panic(err)
	}
}

func (o *CsvOutput) ReportLatencyResult(result LatencyResult) {
	histo := result.TotalHistogram

	columns := []string{"scenario", "samples", "min_ms", "mean_ms", "max_ms", "stdev", "p50_ms", "p75_ms", "p99_ms", "p99999_ms"}
	row := []float64{
		float64(histo.TotalCount()),
		float64(histo.Min()) / 1000.0,
		histo.Mean() / 1000.0,
		float64(histo.Max()) / 1000.0,
		histo.StdDev() / 1000.0,
		float64(histo.ValueAtQuantile(50)) / 1000.0,
		float64(histo.ValueAtQuantile(75)) / 1000.0,
		float64(histo.ValueAtQuantile(95)) / 1000.0,
		float64(histo.ValueAtQuantile(99)) / 1000.0,
		float64(histo.ValueAtQuantile(99.999)) / 1000.0,
	}

	s := strings.Builder{}
	separator := ","
	s.WriteString(strings.Join(columns, separator))
	s.WriteString("\n")

	s.WriteString(fmt.Sprintf("\"%s\"", result.Scenario))

	for i, cell := range row {
		if i > 0 {
			s.WriteString(separator)
		}
		s.WriteString(fmt.Sprintf("%.03f", cell))
	}
	s.WriteString("\n")

	_, err := fmt.Fprint(o.OutStream, s.String())
	if err != nil {
		panic(err)
	}

}

func (o *CsvOutput) Errorf(format string, a ...interface{}) {
	_, err := fmt.Fprintf(o.ErrStream, "ERROR: %s\n", fmt.Sprintf(format, a...))
	if err != nil {
		panic(err)
	}
}
