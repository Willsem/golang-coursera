package main

import (
	"sort"
	"strings"
	"sync"
)

const (
	maxWorkers = 100
	th         = 6
)

func ExecutePipeline(jobs ...job) {
	in := make(chan interface{}, maxWorkers)
	wg := &sync.WaitGroup{}

	for _, job := range jobs {
		out := make(chan interface{}, maxWorkers)
		wg.Add(1)
		go jobWorker(job, in, out, wg)
		in = out
	}

	wg.Wait()
}

func jobWorker(job job, in, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(out)
	job(in, out)
}

func SingleHash(in, out chan interface{}) {
}

func MultiHash(in, out chan interface{}) {
}

func CombineResults(in, out chan interface{}) {
	var result []string

	for data := range in {
		dataString, err := data.(string)

		if !err {
			panic("CombineResults: data from input not a string")
		}

		result = append(result, dataString)
	}

	sort.Strings(result)
	out <- strings.Join(result, "_")
}
