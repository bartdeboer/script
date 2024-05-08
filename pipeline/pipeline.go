package pipeline

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// A pipeable process that doesn't write errors
type Process func(stdin io.Reader, stdout io.Writer) error

// A pipeable process that writes errors
type ProcessE func(stdin io.Reader, stdout io.Writer, stderr io.Writer) error

// Pipeline represents a pipeline object with an associated [ReadAutoCloser].
type Pipeline struct {
	// Reader is the underlying reader.
	Reader         ReadAutoCloser
	stdout, stderr io.Writer
	exitStatus     int

	// because pipe stages are concurrent, protect 'err'
	mu  *sync.Mutex
	err error
}

// NewPipeline creates a new pipe with an empty reader (use [Pipe.WithReader] to
// attach another reader to it).
func NewPipeline() *Pipeline {
	return &Pipeline{
		Reader: ReadAutoCloser{},
		mu:     new(sync.Mutex),
		stdout: os.Stdout,
		stderr: io.Discard,
	}
}

// Bytes returns the contents of the pipe as a []byte, or an error.
func (p *Pipeline) Bytes() ([]byte, error) {
	// if p.Error() != nil {
	// 	return nil, p.Error()
	// }
	data, err := io.ReadAll(p)
	if err != nil {
		p.SetError(err)
	}
	return data, p.Error()
}

// Close closes the pipe's associated reader. This is a no-op if the reader is
// not an [io.Closer].
func (p *Pipeline) Close() error {
	return p.Reader.Close()
}

// Error returns any error present on the pipe, or nil otherwise.
func (p *Pipeline) Error() error {
	if p.mu == nil { // uninitialised pipe
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

var exitStatusPattern = regexp.MustCompile(`exit status (\d+)$`)

// ExitStatus returns the integer exit status of a previous command (for
// example run by [Pipe.Exec]). This will be zero unless the pipe's error
// status is set and the error matches the pattern “exit status %d”.
func (p *Pipeline) ExitStatus() int {
	if p.Error() == nil {
		return 0
	}
	err := p.Error()
	if exitError, ok := err.(*exec.ExitError); ok {
		return exitError.ExitCode()
	}
	match := exitStatusPattern.FindStringSubmatch(err.Error())
	if len(match) < 2 {
		return 0
	}
	status, err := strconv.Atoi(match[1])
	if err != nil {
		// This seems unlikely, but...
		return 0
	}
	return status
}

// Pipe (Filter) adds a new program to the pipeline and ties the output stream [io.Reader]
// of the previous program into the input stream [io.Reader] of the next program.
// It then reads the output stream [io.Writer] of the next program to be used
// for further processing within the pipeline.
//
// Pipe runs concurrently, so its goroutine will not exit until the pipe has
// been fully read. Use [Pipe.Wait] to wait for all concurrent pipes to
// complete.
func (p *Pipeline) Pipe(program ProcessE) *Pipeline {
	started := make(chan bool)
	stdin := p.Reader
	nextStdin, stdout := io.Pipe()
	p = p.WithReader(nextStdin)
	go func() {
		defer close(started)
		defer stdout.Close()
		started <- true
		err := program(stdin, stdout, p.stderr)
		if err != nil {
			p.SetError(err)
		}
	}()
	<-started // Ensure go routines start in sequence
	return p
}

// PipeE wraps programs in WithErr() and writes their error to stderr
func (p *Pipeline) PipeE(program Process) *Pipeline {
	return p.Pipe(WithErr(program))
}

// WithErr wraps the program and automatically writes errors to stderr
func WithErr(program Process) ProcessE {
	return func(stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
		err := program(stdin, stdout)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
		}
		return err
	}
}

// Scanner (FilterScan) sends the contents of the pipe to the function filter, a line at
// a time, and produces the result. filter takes each line as a string and an
// [io.Writer] to write its output to. See [Pipe.Filter] for concurrency
// handling.
func (p *Pipeline) Scanner(filter func(string, io.Writer)) *Pipeline {
	return p.PipeE(Scanner(filter))
}

func Scanner(filter func(string, io.Writer)) Process {
	return func(r io.Reader, w io.Writer) error {
		scanner := newScanner(r)
		for scanner.Scan() {
			filter(scanner.Text(), w)
		}
		return scanner.Err()
	}
}

// Read reads up to len(b) bytes from the pipe into b. It returns the number of
// bytes read and any error encountered. At end of file, or on a nil pipe, Read
// returns 0, [io.EOF].
func (p *Pipeline) Read(b []byte) (int, error) {
	// if p.Error() != nil {
	// 	return 0, p.Error()
	// }
	if p == nil {
		return 0, io.EOF
	}
	return p.Reader.Read(b)
}

// SetError sets the error err on the pipe.
func (p *Pipeline) SetError(err error) *Pipeline {
	if p.mu == nil { // uninitialised pipe
		return p
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.err = err
	return p
}

// Slice returns the pipe's contents as a slice of strings, one element per
// line, or an error.
//
// An empty pipe will produce an empty slice. A pipe containing a single empty
// line (that is, a single \n character) will produce a slice containing the
// empty string as its single element.
func (p *Pipeline) Slice() ([]string, error) {
	result := []string{}
	p.Scanner(func(line string, w io.Writer) {
		result = append(result, line)
	}).Wait()
	return result, p.Error()
}

// Stdout copies the pipe's contents to its configured standard output (using
// [Pipe.WithStdout]), or to [os.Stdout] otherwise, and returns the number of
// bytes successfully written, together with any error.
func (p *Pipeline) Stdout() (int, error) {
	// if p.Error() != nil {
	// 	return 0, p.Error()
	// }
	n64, err := io.Copy(p.stdout, p)
	if err != nil {
		return 0, err
	}
	n := int(n64)
	if int64(n) != n64 {
		return 0, fmt.Errorf("length %d overflows int", n64)
	}
	return n, p.Error()
}

// String returns the pipe's contents as a string, together with any error.
func (p *Pipeline) String() (string, error) {
	data, err := p.Bytes()
	if err != nil {
		p.SetError(err)
	}
	return string(data), p.Error()
}

// Int returns the pipe's contents as an int64, together with any error.
func (p *Pipeline) Int64() (int64, error) {
	data, err := p.Bytes()
	if err != nil {
		p.SetError(err)
	}
	strData := strings.TrimSpace(string(data))
	result, convErr := strconv.ParseInt(strData, 10, 64)
	if convErr != nil {
		return 0, convErr
	}
	return result, p.Error()
}

// Int returns the pipe's contents as an int, together with any error.
func (p *Pipeline) Int() (int, error) {
	result64, err := p.Int64()
	return int(result64), err
}

// Wait reads the pipe to completion and discards the result. This is mostly
// useful for waiting until concurrent filters have completed (see
// [Pipe.Filter]).
func (p *Pipeline) Wait() *Pipeline {
	// Drain the pipeline to ensure all writers can finish writing
	_, err := io.Copy(io.Discard, p)
	if err != nil {
		p.SetError(err)
	}
	return p
}

// WithError sets the error err on the pipe.
func (p *Pipeline) WithError(err error) *Pipeline {
	p.SetError(err)
	return p
}

// WithReader sets the pipe's input reader to r. Once r has been completely
// read, it will be closed if necessary.
func (p *Pipeline) WithReader(r io.Reader) *Pipeline {
	p.Reader = NewReadAutoCloser(r)
	return p
}

// WithStderr redirects the standard error output for commands run via
// [Pipe.Exec] or [Pipe.ExecForEach] to the writer w, instead of going to the
// pipe as it normally would.
func (p *Pipeline) WithStderr(w io.Writer) *Pipeline {
	p.stderr = w
	return p
}

// WithStdout sets the pipe's standard output to the writer w, instead of the
// default [os.Stdout].
func (p *Pipeline) WithStdout(w io.Writer) *Pipeline {
	p.stdout = w
	return p
}

// ReadAutoCloser wraps an [io.ReadCloser] so that it will be automatically
// closed once it has been fully read.
type ReadAutoCloser struct {
	r        io.ReadCloser
	isClosed bool
}

// NewReadAutoCloser returns a [ReadAutoCloser] wrapping the reader r.
func NewReadAutoCloser(r io.Reader) ReadAutoCloser {
	if _, ok := r.(io.Closer); !ok {
		return ReadAutoCloser{io.NopCloser(r), false}
	}
	rc, ok := r.(io.ReadCloser)
	if !ok {
		// This can never happen, but just in case it does...
		panic("internal error: type assertion to io.ReadCloser failed")
	}
	return ReadAutoCloser{rc, false}
}

// Close closes ra's reader, returning any resulting error.
func (ra ReadAutoCloser) Close() error {
	if ra.r == nil {
		return nil
	}
	ra.isClosed = true
	return ra.r.Close()
}

// Read reads up to len(b) bytes from ra's reader into b. It returns the number
// of bytes read and any error encountered. At end of file, Read returns 0,
// [io.EOF]. If end-of-file is reached, the reader will be closed.
func (ra ReadAutoCloser) Read(b []byte) (n int, err error) {
	if ra.r == nil {
		return 0, io.EOF
	}
	n, err = ra.r.Read(b)
	if ra.isClosed {
		err = io.EOF
	} else if err == io.EOF {
		ra.Close()
	}
	return n, err
}

func newScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), math.MaxInt)
	return scanner
}
