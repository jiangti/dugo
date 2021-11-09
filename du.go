package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"

	"github.com/alitto/pond"
)

type pathObject struct {
	currPath string
	info     os.FileInfo
	depth    int
}

func goDiskUsage(pathChan chan pathObject) int64 {
	for {
		path := <-pathChan
		return diskUsage(path.currPath, path.info, path.depth)
	}

}

func diskUsage(currPath string, info os.FileInfo, depth int) int64 {
	var size int64

	if *excludes != "" {
		matched, err := regexp.MatchString(*excludes, currPath)

		if err != nil {

			log.Fatal(err)
		}

		if matched {
			return size

		}
	}

	dir, err := os.Open(currPath)
	if err != nil {
		fmt.Println(err)
		return size
	}

	defer dir.Close()

	files, err := dir.Readdir(-1)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	pool := pond.New(5, 10)

	for _, file := range files {
		if file.IsDir() {

			nextPath := fmt.Sprintf("%s/%s", currPath, file.Name())

			if (*maxDepth) <= 0 || (*maxDepth) < depth {
				if threshold == 0 || size >= threshold {
					pool.Submit(func() {
						size += diskUsage(nextPath, file, depth+1)
					})
				}
			} else {
				size += diskUsage(nextPath, file, depth+1)
			}

		} else {
			size += file.Size()
		}
	}

	pool.StopAndWait()

	if (*maxDepth) <= 0 || (*maxDepth) >= depth {
		if threshold == 0 || size >= threshold {
			prettyPrintSize(size)
			fmt.Printf("\t %s%c\n", currPath, filepath.Separator)
		}
	}

	return size
}

func prettyPrintSize(size int64) {
	if *humanReadable {
		switch {
		case size > 1024*1024*1024:
			fmt.Printf("%.1fG", float64(size)/(1024*1024*1024))
		case size > 1024*1024:
			fmt.Printf("%.1fM", float64(size)/(1024*1024))
		case size > 1024:
			fmt.Printf("%.1fK", float64(size)/1024)
		default:
			fmt.Printf("%d", size)
		}
	} else {
		fmt.Printf("%d", size)
	}
}

var usage = `Usage: dugo [options...] <target_dir>

Options:
  -h  "Human-readable" output.  Use unit suffixes: Byte, Kilobyte, Megabyte, Gigabyte.
  -t  threshold of the size, any folders' size larger than the threshold will be print. for example, '1G', '10M', '100K', '1024'
  -d  list its subdirectories and their sizes to any desired level of depth (i.e., to any level of subdirectories) in a directory tree.
  -v  excludes folder(s) (i.e. app,.git,node_modules).
`

var (
	humanReadable = flag.Bool("h", false, "human readable unit of size")
	thresholdStr  = flag.String("t", "", "the threshold for printing the folder size")
	threshold     int64
	maxDepth      = flag.Int("d", 0, "list its subdirectories and their sizes to any desired level of depth (i.e., to any level of subdirectories) in a directory tree.")
	excludes      = flag.String("v", "", "excludes folder(s) (i.e. app,.git,node_modules)")
)

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(usage, runtime.NumCPU()))
	}

	pathChan := make(chan pathObject)

	for i := 0; i < 10; i++ {
		go goDiskUsage(pathChan)
	}

	flag.Parse()

	l := len(*thresholdStr)
	if l > 0 {
		t, err := strconv.Atoi(*thresholdStr)
		if err != nil { // threshold string ends with a unit char
			i, err := strconv.Atoi((*thresholdStr)[0:(l - 1)])
			if err != nil {
				usageAndExit("")
			}

			switch (*thresholdStr)[l-1:] {
			case "G":
				t = i * 1024 * 1024 * 1024
			case "M":
				t = i * 1024 * 1024
			case "K":
				t = i * 1024
			}
		}
		threshold = int64(t)
	}

	dir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if flag.NArg() > 0 {
		dir = flag.Args()[0]
	}

	info, err := os.Lstat(dir)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	diskUsage(dir, info, 0)
}

func usageAndExit(msg string) {
	if msg != "" {
		fmt.Fprintf(os.Stderr, msg)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}
