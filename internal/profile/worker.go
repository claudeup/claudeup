// ABOUTME: Worker pool for concurrent job execution during profile apply
// ABOUTME: Executes jobs in parallel with configurable worker count
package profile

import (
	"sync"
)

// DefaultWorkers is the default number of concurrent workers
const DefaultWorkers = 4

// Job represents a unit of work to execute
type Job struct {
	Name    string       // Identifier for the job (e.g., plugin name)
	Type    string       // Job type (e.g., "marketplace", "plugin", "mcp")
	Execute func() error // The work to perform
}

// JobResult represents the outcome of executing a job
type JobResult struct {
	Name    string // Job identifier
	Type    string // Job type
	Success bool   // True if Execute returned nil
	Error   error  // The error if Execute failed
}

// RunWorkerPool executes jobs concurrently using a worker pool
// Returns results in completion order (not input order)
func RunWorkerPool(jobs []Job, workers int) []JobResult {
	if len(jobs) == 0 {
		return nil
	}

	if workers <= 0 {
		workers = DefaultWorkers
	}

	// Channels for job distribution and result collection
	jobsChan := make(chan Job, len(jobs))
	resultsChan := make(chan JobResult, len(jobs))

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobsChan {
				result := JobResult{
					Name: job.Name,
					Type: job.Type,
				}
				if err := job.Execute(); err != nil {
					result.Error = err
					result.Success = false
				} else {
					result.Success = true
				}
				resultsChan <- result
			}
		}()
	}

	// Feed jobs to workers
	for _, job := range jobs {
		jobsChan <- job
	}
	close(jobsChan)

	// Wait for all workers to finish, then close results
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect all results
	results := make([]JobResult, 0, len(jobs))
	for result := range resultsChan {
		results = append(results, result)
	}

	return results
}

// RunWorkerPoolWithCallback executes jobs and calls callback after each completion
// Useful for updating progress UI as jobs complete
func RunWorkerPoolWithCallback(jobs []Job, workers int, callback func(JobResult)) []JobResult {
	if len(jobs) == 0 {
		return nil
	}

	if workers <= 0 {
		workers = DefaultWorkers
	}

	jobsChan := make(chan Job, len(jobs))
	resultsChan := make(chan JobResult, len(jobs))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobsChan {
				result := JobResult{
					Name: job.Name,
					Type: job.Type,
				}
				if err := job.Execute(); err != nil {
					result.Error = err
					result.Success = false
				} else {
					result.Success = true
				}
				resultsChan <- result
			}
		}()
	}

	// Feed jobs
	for _, job := range jobs {
		jobsChan <- job
	}
	close(jobsChan)

	// Collect results with callback
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	results := make([]JobResult, 0, len(jobs))
	for result := range resultsChan {
		if callback != nil {
			callback(result)
		}
		results = append(results, result)
	}

	return results
}
