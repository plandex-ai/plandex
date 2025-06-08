package handlers

import (
	"context"
	"errors"
	"log"
	"math"
	"plandex-server/syntax/file_map"
	shared "plandex-shared"
	"runtime"
	"sync"
	"time"
)

// simple in-memory per-instance queue for file map jobs
// ensures mapping doesn't take over all available CPUs

const fileMapMaxQueueSize = 20 // caller errors out if this is exceeded
var fileMapMaxConcurrency = 3  // set to 3/4 of available CPUs below
const mapJobTimeout = 60 * time.Second

type projectMapJob struct {
	inputs  shared.FileMapInputs
	ctx     context.Context
	results chan shared.FileMapBodies
}

var projectMapQueue = make(chan projectMapJob, fileMapMaxQueueSize)

var mapCPUSem chan struct{}

func init() {
	// Use 3/4 of available CPUs for mapping workers
	cpus := runtime.NumCPU()
	fileMapMaxConcurrency = int(math.Ceil(float64(cpus) * 0.75))
	if fileMapMaxConcurrency < 1 {
		fileMapMaxConcurrency = 1
	}

	log.Printf("fileMapMaxConcurrency: %d", fileMapMaxConcurrency)

	mapCPUSem = make(chan struct{}, fileMapMaxConcurrency)

	// start workers, one per CPU
	for i := 0; i < fileMapMaxConcurrency; i++ {
		go processProjectMapQueue()
	}
}

func processProjectMapQueue() {
	for job := range projectMapQueue {
		if job.ctx.Err() != nil {
			if job.ctx.Err() == context.DeadlineExceeded {
				log.Printf("processProjectMapQueue: job context deadline exceeded: %v", job.ctx.Err())
				safeSend(job.results, nil)
				continue
			}
			log.Printf("processProjectMapQueue: job context cancelled: %v", job.ctx.Err())
			safeSend(job.results, nil)
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
	log.Printf("queueProjectMapJob: len(projectMapQueue): %d", len(projectMapQueue))
	select {
	case projectMapQueue <- job:
		return nil
	default:
		return errors.New("queue is full")
	}
}

func mapWorker(job projectMapJob) {
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
		go func(path string, input string) {
			if job.ctx.Err() != nil {
				wg.Done()
				return
			}

			mapCPUSem <- struct{}{}
			defer func() { <-mapCPUSem }()
			defer wg.Done()

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
		safeSend(job.results, nil)
		return
	}

	safeSend(job.results, maps)
}

func safeSend(ch chan shared.FileMapBodies, v shared.FileMapBodies) {
	// never block, never panic
	select {
	case ch <- v:
	default: // buffer already full – receiver must have gone away
	}
}
