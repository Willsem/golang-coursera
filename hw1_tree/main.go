package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	//"path/filepath"
	//"strings"
)

type Stack struct {
	filename []string
	isdir    []bool
	size     []int64
	length   int
}

func (s *Stack) Init() {
	s.filename = make([]string, 0, 10)
	s.isdir = make([]bool, 0, 10)
	s.size = make([]int64, 0, 10)
	s.length = 0
}

func (s *Stack) Push(el string, isdir bool, size int64) {
	s.filename = append(s.filename, el)
	s.isdir = append(s.isdir, isdir)
	s.size = append(s.size, size)
	s.length++
}

func (s *Stack) Pop() (string, bool, int64) {
	el := s.filename[s.length-1]
	s.filename = s.filename[:len(s.filename)-1]

	isdir := s.isdir[s.length-1]
	s.isdir = s.isdir[:len(s.isdir)-1]

	size := s.size[s.length-1]
	s.size = s.size[:len(s.size)-1]

	s.length--

	return el, isdir, size
}

func (s Stack) IsEmpty() bool {
	return s.length == 0
}

func (s Stack) NextIsDotDot() bool {
	return s.filename[s.length-1] == ".."
}

func Cd(path string) (file *os.File, err error) {
	file, err = os.Open(path)
	if err != nil {
		return
	}
	if err = file.Chdir(); err != nil {
		return
	}

	file, err = os.Open(".")
	if err != nil {
		return
	}
	if err = file.Chdir(); err != nil {
		return
	}

	return
}

func printLines(out io.Writer, level int, probels []bool, end bool) {
	for i := 0; i < level-1; i++ {
		if probels[i] {
			fmt.Fprint(out, "│\t")
		} else {
			fmt.Fprint(out, "\t")
		}
	}

	if end {
		fmt.Fprint(out, "└───")
	} else {
		fmt.Fprint(out, "├───")
	}
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	var stack Stack
	stack.Init()

	file, err := Cd(path)

	var level int = 0
	probels := make([]bool, 0, 10)
	stack.Push(".", true, 0)

	for !stack.IsEmpty() {
		filename, isdir, size := stack.Pop()

		if !isdir {
			printLines(out, level, probels, stack.NextIsDotDot())
			if size == 0 {
				fmt.Fprintf(out, "%s (%s)\n", filename, "empty")
			} else {
				fmt.Fprintf(out, "%s (%db)\n", filename, size)
			}

			continue
		}

		file, err = Cd(filename)
		if err != nil {
			return err
		}

		if filename == ".." {
			file.Close()
			level--
			probels = probels[:len(probels)-1]
			continue
		}

		if filename != "." {
			probels[level-1] = !stack.NextIsDotDot()
			printLines(out, level, probels, stack.NextIsDotDot())
			fmt.Fprintln(out, filename)
		}

		stack.Push("..", true, 0)

		names, err := file.Readdir(-1)
		if err != nil {
			return err
		}

		for i := 0; i < len(names); i++ {
			for j := 0; j < len(names)-i-1; j++ {
				if sort.StringsAreSorted([]string{names[j].Name(), names[j+1].Name()}) {
					names[j], names[j+1] = names[j+1], names[j]
				}
			}
		}

		level++
		probels = append(probels, true)
		for _, file := range names {
			if file.IsDir() {
				stack.Push(file.Name(), true, 0)
			} else if printFiles {
				stack.Push(file.Name(), false, file.Size())
			}
		}

		file.Close()
	}

	return nil
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
