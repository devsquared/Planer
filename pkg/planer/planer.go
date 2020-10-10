package planer

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"sync"
	"time"
)

//TODO: need to go through and test. Wrote this patch set in a quick idea period.

// PlaneLog is the main entry point for the planer app
func PlaneLog(from time.Time, to time.Time, searchedWord string, fileName string) (string, error) {
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
		process(file, from, to, searchedWord)
	}

	message = "Time taken - " + string(time.Since(s))
	return message, nil
}

func process(f *os.File, start time.Time, end time.Time, word string) ([]string, error) {
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

	var allStrings []string
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

			return nil, err
		}

		nextUntilNewline, err := r.ReadBytes('\n')

		if err != io.EOF {
			buf = append(buf, nextUntilNewline...)
		}

		waitGroup.Add(1)
		go func() {
			allStrings = append(allStrings, processChunk(buf, &linesPool, &stringPool, start, end, word)...)
			waitGroup.Done()
		}()
	}

	waitGroup.Wait()
	return allStrings, nil
}

func processChunk(chunk []byte, linesPool *sync.Pool, stringPool *sync.Pool, start time.Time, end time.Time, word string) []string {

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

	var allFilteredStrings []string
	for i := 0; i < (numOfThreads); i++ {
		waitGroup2.Add(1)
		strings := make(chan string)
		go conProcessChunks(&waitGroup2, logsSlice, word, i*chunkSize, int(math.Min(float64((i+1)*chunkSize), float64(len(logsSlice)))), strings, start, end)
		allFilteredStrings = append(allFilteredStrings, <-strings)
	}

	waitGroup2.Wait()
	logsSlice = nil

	return allFilteredStrings
}

func conProcessChunks(waitGroup *sync.WaitGroup, logsSlice []string, word string, s int, e int, channel chan string, start time.Time, end time.Time) {
	// here we concurrently process the chunks and pass back
	// the chunk that fits the criteria back through the given channel
	defer waitGroup.Done()
	var stringBuilder strings.Builder
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

		if logCreationTime.After(start) && logCreationTime.Before(end) && strings.Contains(text, word) {
			// TODO: refactor in order to output a built string of all filtered lines
			stringBuilder.WriteString(text)
		}
	}

	channel <- stringBuilder.String()
}
