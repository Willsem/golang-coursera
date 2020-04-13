package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	in := make(chan interface{}, 100)

	wg := &sync.WaitGroup{}

	for _, job := range jobs {
		wg.Add(1)
		out := make(chan interface{}, 100)
		go worker(wg, job, in, out)
		in = out
	}

	wg.Wait()
}

func worker(wg *sync.WaitGroup, job job, in, out chan interface{}) {
	defer wg.Done()
	defer close(out)
	job(in, out)
}

func SingleHash(in, out chan interface{}) {
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	for input := range in {
		wg.Add(1)
		go singleHashWorker(wg, mu, input, out)
	}

	wg.Wait()
}

func singleHashWorker(wg *sync.WaitGroup, mu *sync.Mutex, input interface{}, out chan interface{}) {
	defer wg.Done()

	in, err := input.(int)
	if !err {
		panic("SingleHash : input is not a int")
	}

	data := strconv.Itoa(in)

	mu.Lock()
	md5Data := DataSignerMd5(data)
	mu.Unlock()

	crc32Chan := make(chan string)

	go crc32SignerWorker(data, crc32Chan)

	crc32Md5Data := DataSignerCrc32(md5Data)
	crc32Data := <-crc32Chan

	out <- crc32Data + "~" + crc32Md5Data
}

func crc32SignerWorker(data string, out chan string) {
	out <- DataSignerCrc32(data)
}

func MultiHash(in, out chan interface{}) {
	const th = 6
	wg := &sync.WaitGroup{}

	for input := range in {
		wg.Add(1)
		go multiHashWorker(wg, input, out, th)
	}

	wg.Wait()
}

func multiHashWorker(wg *sync.WaitGroup, input interface{}, out chan interface{}, th int) {
	defer wg.Done()

	mu := &sync.Mutex{}
	wgJob := &sync.WaitGroup{}
	result := make([]string, th)

	in, err := input.(string)
	if !err {
		panic("MultiHash : input is not a string")
	}

	for i := 0; i < th; i++ {
		wgJob.Add(1)
		data := strconv.Itoa(i) + in

		go func(res []string, index int, data string) {
			defer wgJob.Done()
			data = DataSignerCrc32(data)

			mu.Lock()
			res[index] = data
			mu.Unlock()
		}(result, i, data)
	}

	wgJob.Wait()
	out <- strings.Join(result, "")
}

func CombineResults(in, out chan interface{}) {
	var result []string

	for input := range in {
		data, err := input.(string)
		if !err {
			panic("CombineResults : input is not a string")
		}

		result = append(result, data)
	}

	sort.Strings(result)
	out <- strings.Join(result, "_")
}
