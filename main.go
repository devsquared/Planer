package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

func main() {
	// TODO: typing a timestamp in a cli seems awkward, should give a yaml config to ease this; then look at a wails app
	// TODO: create multiple methods to allow for word search in log
	// TODO: provide a file structure to drop in log files and get output files for saving
	// TODO: provide way to define the time format
	// TODO: look at merging with the file search project
	if err := cli(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func cli() error {
	// setup a ctrl-c interrupt to cancel and exit the program
	cancelled := setupCloseHandler()

	fmt.Println("Welcome to Planer, the easy way to make your logs manageable.")
	fmt.Println("-------------------------------------------------------------")

	// TODO: when we implement term filtering, we will ask if the user wants this
	// and what the term is

	// prompt a user for the args we need
	fmt.Println("To filter down the log, what is the 'from' portion of the time range?")
	fmt.Println("(Format the timestamp as such - 2006-01-02T15:04:05.0000Z")

	fromString, err := getInputText()
	if err != nil {
		fmt.Println("Issue getting from timestamp input text.")
		return err
	}

	from, err := time.Parse("2006-01-02T15:04:05.0000Z", fromString)
	if err != nil {
		fmt.Println("Unable to parse the from time arg : ", fromString)
		return err
	}

	fmt.Println("What is the 'to' portion of the time range? Alternatively, leave arg blank to default to now.")
	fmt.Println("(Format the timestamp as such - 2006-01-02T15:04:05.0000Z")

	toString, err := getInputText()
	if err != nil {
		fmt.Println("Issue getting the to timestamp input text.")
		return err
	}

	var to time.Time
	if toString == "" {
		to = time.Now()
	} else {
		to, err = time.Parse("2006-01-02T15:04:05.0000Z", toString)
		if err != nil {
			fmt.Println("Unable to parse the to time arg : ", toString)
			return err
		}
	}

	fmt.Println("What is the log file to plane down?")
	fmt.Println("Please provide a full path.")

	fileName, err := getInputText()
	if err != nil {
		fmt.Println("Issue getting the fileName input text.")
		return err
	}

	if !cancelled {
		message, err := planeLog(from, to, fileName)
		if err != nil {
			return err
		}

		fmt.Println(message)
	}

	return nil
}

func planeLog(from time.Time, to time.Time, fileName string) (string, error) {
	var message string

	s := time.Now()

	file, err := os.Open(fileName)
	if err != nil {
		message = "Unable to open file : " + fileName
		return message, err
	}

	defer file.Close()

	filestat, err := file.Stat()
	if err != nil {
		message = "Could not get the file stat"
		return message, err
	}

	fileSize := filestat.Size()
	offset := fileSize - 1
	lastLineSize := 0

	for {
		b := make([]byte, 1)
		n, err := file.ReadAt(b, offset)
		if err != nil {
			fmt.Println("Error reading file : ", err) // should output a line here
			break
		}

		char := string(b[0])
		if char == "\n" {
			break
		}

		offset--
		lastLineSize += n
	}

	lastLine := make([]byte, lastLineSize)
	_, err = file.ReadAt(lastLine, offset+1)
	if err != nil {
		message = "Could not read last line with offset, " + string(offset) + " and lastLineSize, " + string(lastLineSize)
		return message, err
	}

	logSlice := strings.SplitN(string(lastLine), ",", 2)
	logCreationTimeString := logSlice[0]

	lastLogCreationTime, err := time.Parse("2006-01-02T15:04:05.0000Z", logCreationTimeString)
	if err != nil {
		message = "Unable to parse time : " + logCreationTimeString
		return message, err
	}

	if lastLogCreationTime.After(from) && lastLogCreationTime.Before(to) {
		Process(file, from, to)
	}

	message = "Time taken - " + string(time.Since(s))
	return message, nil
}

func getInputText() (string, error) {
	// prompt for input
	fmt.Print(">> ")

	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	text = strings.TrimSuffix(text, "\n")

	return text, nil
}

// NOTE: This close handler does not seem to work in situations where a IDE handles the running of the app
// It, however, works as intended in a terminal.
func setupCloseHandler() bool {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() bool {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal")
		os.Exit(0)
		return true
	}()

	//base case
	return false
}

func Process(f *os.File, start time.Time, end time.Time) error {
	linesPool := sync.Pool{New: func() interface{} {
		lines := make([]byte, 250*1024)
		return lines
	}}

	stringPool := sync.Pool{New: func() interface{} {
		lines := ""
		return lines
	}}

	r := bufio.NewReader(f)

	var waitGroup sync.WaitGroup

	for {
		buf := linesPool.Get().([]byte)

		n, err := r.Read(buf)
		buf = buf[:n]

		if n == 0 {
			if err != nil {
				fmt.Println(err)
				break
			}
			if err == io.EOF {
				break
			}

			return err
		}

		nextUntilNewline, err := r.ReadBytes('\n')

		if err != io.EOF {
			buf = append(buf, nextUntilNewline...)
		}

		waitGroup.Add(1)
		go func() {
			ProcessChunk(buf, &linesPool, &stringPool, start, end)
			waitGroup.Done()
		}()
	}

	waitGroup.Wait()
	return nil
}

func ProcessChunk(chunk []byte, linesPool *sync.Pool, stringPool *sync.Pool, start time.Time, end time.Time) {

	var waitGroup2 sync.WaitGroup

	logs := stringPool.Get().(string)
	logs = string(chunk)

	linesPool.Put(chunk)

	logsSlice := strings.Split(logs, "\n")

	stringPool.Put(logs)

	chunkSize := 300
	n := len(logsSlice)
	numOfThreads := n / chunkSize

	if n%chunkSize != 0 {
		numOfThreads++
	}

	for i := 0; i < (numOfThreads); i++ {
		waitGroup2.Add(1)
		go func(s int, e int) {
			defer waitGroup2.Done()
			for i := s; i < e; i++ {
				text := logsSlice[i]
				if len(text) == 0 {
					continue
				}

				logSlice := strings.SplitN(text, ",", 2)
				logCreationTimeString := logSlice[0]

				logCreationTime, err := time.Parse("2006-01-02T15:04:05.0000Z", logCreationTimeString)
				if err != nil {
					fmt.Println("Unable to parse time : " + logCreationTimeString + " for the log : " + text)
					return
				}

				if logCreationTime.After(start) && logCreationTime.Before(end) {
					fmt.Println(text)
				}
			}
		}(i*chunkSize, int(math.Min(float64((i+1)*chunkSize), float64(len(logsSlice)))))
	}

	waitGroup2.Wait()
	logsSlice = nil
}
