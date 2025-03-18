package handlers

import (
	"context"
	"errors"
	"log"
	"plandex-server/syntax/file_map"
	shared "plandex-shared"
	"runtime"
	"sync"
	"time"
)

// simple in-memory per-instance queue for file map jobs
// ensures mapping doesn't take over all available CPUs

const fileMapMaxQueueSize = 20 // caller errors out if this is exceeded
var fileMapMaxConcurrency = 2  // set to half of available CPUs below
const mapJobTimeout = 60 * time.Second

type projectMapJob struct {
	inputs  shared.FileMapInputs
	ctx     context.Context
	results chan shared.FileMapBodies
}

var projectMapQueue = make(chan projectMapJob, fileMapMaxQueueSize)

func init() {
	// Use half of available CPUs for mapping workers
	cpus := runtime.NumCPU()
	fileMapMaxConcurrency = cpus / 2
	if fileMapMaxConcurrency < 1 {
		fileMapMaxConcurrency = 1
	}

	go processProjectMapQueue()
}

// map jobs are processed serially, one at a time
// the jobs themselves can use concurrency
func processProjectMapQueue() {
	for job := range projectMapQueue {
		if job.ctx.Err() != nil {
			log.Printf("processProjectMapQueue: job context cancelled: %v", job.ctx.Err())
			continue
		}
		ctxWithTimeout, cancel := context.WithTimeout(job.ctx, mapJobTimeout)
		mapWorker(projectMapJob{
			inputs:  job.inputs,
			ctx:     ctxWithTimeout,
			results: job.results,
		})
		cancel()
	}
}

func queueProjectMapJob(job projectMapJob) error {
	select {
	case projectMapQueue <- job:
		return nil
	default:
		return errors.New("queue is full")
	}
}

func mapWorker(job projectMapJob) {
	sem := make(chan struct{}, fileMapMaxConcurrency)
	maps := make(shared.FileMapBodies)
	wg := sync.WaitGroup{}
	var mu sync.Mutex

	log.Printf("mapWorker: len(job.inputs): %d", len(job.inputs))

	for path, input := range job.inputs {
		if !shared.HasFileMapSupport(path) {
			mu.Lock()
			maps[path] = "[NO MAP]"
			mu.Unlock()
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(path string, input string) {
			defer wg.Done()
			defer func() { <-sem }()
			if job.ctx.Err() != nil {
				return
			}
			fileMap, err := file_map.MapFile(job.ctx, path, []byte(input))
			if err != nil {
				// Skip files that can't be parsed, just log the error
				log.Printf("Error mapping file %s: %v", path, err)
				mu.Lock()
				maps[path] = "[NO MAP]"
				mu.Unlock()
				return
			}
			mu.Lock()
			maps[path] = fileMap.String()
			mu.Unlock()
		}(path, input)
	}

	wg.Wait()

	if job.ctx.Err() != nil {
		job.results <- nil
		return
	}

	job.results <- maps
}
