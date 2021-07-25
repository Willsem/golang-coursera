package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	maxWorkers = 100
	thCount    = 6
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
	wg := &sync.WaitGroup{}
	quota := make(chan struct{}, 1)

	for data := range in {
		dataInt, err := data.(int)
		if !err {
			panic("SingleHash: input data is not an int")
		}

		dataString := strconv.Itoa(dataInt)

		wg.Add(1)
		go singleHashWorker(dataString, out, wg, quota)
	}

	wg.Wait()
}

func singleHashWorker(data string, out chan<- interface{}, wg *sync.WaitGroup, quota chan struct{}) {
	defer wg.Done()

	outCrc32 := make(chan string)
	outMd5 := make(chan string)
	outCrc32AfterMd5 := make(chan string)

	go crc32Worker(data, outCrc32)
	go md5Worker(data, outMd5, quota)

	dataMd5 := <-outMd5
	go crc32Worker(dataMd5, outCrc32AfterMd5)

	part1 := <-outCrc32
	part2 := <-outCrc32AfterMd5
	out <- part1 + "~" + part2
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}

	for data := range in {
		dataString, err := data.(string)
		if !err {
			panic("MultiHash: input data is not a string")
		}

		wg.Add(1)
		go multiHashWorker(dataString, out, wg)
	}

	wg.Wait()
}

func multiHashWorker(data string, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()

	wgWorkers := &sync.WaitGroup{}
	mu := &sync.Mutex{}
	result := make([]string, thCount)

	for th := 0; th < thCount; th++ {
		dataInput := strconv.Itoa(th) + data

		wgWorkers.Add(1)
		go func(data string, result []string, index int, wg *sync.WaitGroup, mu *sync.Mutex) {
			defer wg.Done()

			ch := make(chan string)
			go crc32Worker(data, ch)

			workerOut := <-ch

			mu.Lock()
			result[index] = workerOut
			mu.Unlock()
		}(dataInput, result, th, wgWorkers, mu)
	}

	wgWorkers.Wait()
	out <- strings.Join(result, "")
}

func CombineResults(in, out chan interface{}) {
	var result []string

	for data := range in {
		dataString, err := data.(string)

		if !err {
			panic("CombineResults: input data is not a string")
		}

		result = append(result, dataString)
	}

	sort.Strings(result)
	out <- strings.Join(result, "_")
}

func crc32Worker(data string, out chan<- string) {
	out <- DataSignerCrc32(data)
}

func md5Worker(data string, out chan<- string, quota chan struct{}) {
	quota <- struct{}{}
	out <- DataSignerMd5(data)
	<-quota
}
