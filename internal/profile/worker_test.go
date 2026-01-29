// ABOUTME: Tests for concurrent worker pool used during profile apply
// ABOUTME: Validates job execution, result collection, and error handling
package profile

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPoolExecutesAllJobs(t *testing.T) {
	var executed int32

	jobs := []Job{
		{Name: "job1", Execute: func() error { atomic.AddInt32(&executed, 1); return nil }},
		{Name: "job2", Execute: func() error { atomic.AddInt32(&executed, 1); return nil }},
		{Name: "job3", Execute: func() error { atomic.AddInt32(&executed, 1); return nil }},
	}

	results := RunWorkerPoolWithCallback(jobs, 2, nil)

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	if int(executed) != 3 {
		t.Errorf("expected 3 jobs executed, got %d", executed)
	}
}

func TestWorkerPoolCollectsErrors(t *testing.T) {
	jobs := []Job{
		{Name: "success", Execute: func() error { return nil }},
		{Name: "failure", Execute: func() error { return fmt.Errorf("intentional error") }},
	}

	results := RunWorkerPoolWithCallback(jobs, 2, nil)

	var successes, failures int
	for _, r := range results {
		if r.Error != nil {
			failures++
		} else {
			successes++
		}
	}

	if successes != 1 {
		t.Errorf("expected 1 success, got %d", successes)
	}
	if failures != 1 {
		t.Errorf("expected 1 failure, got %d", failures)
	}
}

func TestWorkerPoolRunsConcurrently(t *testing.T) {
	// Create jobs that take time - with 4 workers, 4 jobs should complete
	// in roughly 1 job-duration, not 4x
	start := time.Now()
	jobDuration := 50 * time.Millisecond

	jobs := make([]Job, 4)
	for i := range jobs {
		jobs[i] = Job{
			Name:    fmt.Sprintf("job%d", i),
			Execute: func() error { time.Sleep(jobDuration); return nil },
		}
	}

	RunWorkerPoolWithCallback(jobs, 4, nil)

	elapsed := time.Since(start)

	// Should complete in roughly 1 job duration (with some overhead)
	// If running sequentially, would be ~200ms
	maxExpected := jobDuration * 2
	if elapsed > maxExpected {
		t.Errorf("expected concurrent execution in ~%v, took %v", jobDuration, elapsed)
	}
}

func TestWorkerPoolWithZeroJobs(t *testing.T) {
	results := RunWorkerPoolWithCallback(nil, 4, nil)

	if len(results) != 0 {
		t.Errorf("expected 0 results for nil jobs, got %d", len(results))
	}
}

func TestWorkerPoolPreservesJobName(t *testing.T) {
	jobs := []Job{
		{Name: "my-plugin@marketplace", Execute: func() error { return nil }},
	}

	results := RunWorkerPoolWithCallback(jobs, 1, nil)

	if results[0].Name != "my-plugin@marketplace" {
		t.Errorf("expected job name preserved, got %s", results[0].Name)
	}
}
