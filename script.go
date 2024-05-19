package script

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/bartdeboer/pipeline"
	"github.com/bartdeboer/pipeline/std"
	"github.com/bartdeboer/script/gojq"
	"github.com/bartdeboer/script/shell"
)

type Pipe struct {
	std.Pipeline[*Pipe]
	stdout io.Writer

	httpClient *http.Client
}

func NewPipe() *Pipe {
	p := &Pipe{
		httpClient: http.DefaultClient,
	}
	p.Pipeline = std.NewPipeline(p)
	p.WithStdout(os.Stdout)
	p.SetCombinedOutput(true)
	return p
}

// For backwards compatibility
func (p *Pipe) Filter(filter func(r io.Reader, w io.Writer) error) *Pipe {
	b := pipeline.NewBaseProgram()
	b.StartFn = func() error {
		return filter(b.Stdin, b.Stdout)
	}
	return p.Pipe(b)
}

// For backwards compatibility
func (p *Pipe) FilterScan(filter func(string, io.Writer)) *Pipe {
	return p.Scanner(filter)
}

func (p *Pipe) Stdout() (int, error) {
	n64, err := p.Pipeline.Run()
	n := int(n64)
	if int64(n) != n64 {
		return 0, fmt.Errorf("length %d overflows int", n64)
	}
	return n, err
}

// Sources:

// Args creates a pipeline with the command line arguments
func Args() *Pipe {
	return Slice(os.Args[1:])
}

// Do creates a pipeline with an HTTP request
func Do(req *http.Request) *Pipe {
	return NewPipe().Do(req)
}

// Echo creates a pipeline with the specified string
func Echo(s string) *Pipe {
	return NewPipe().Echo(s)
}

// Exec creates a pipeline with the specified command using sh/shell
func Exec(cmdLine string) *Pipe {
	return NewPipe().Exec(cmdLine)
}

// File creates a pipeline with the file contents
func File(path string) *Pipe {
	return NewPipe().Pipe(std.File(path))
}

// FindFiles creates a pipeline with the files found in dir
func FindFiles(dir string) *Pipe {
	return NewPipe().Pipe(std.FindFiles(dir))
}

// Do creates a pipeline with a GET HTTP request
func Get(url string) *Pipe {
	return NewPipe().Get(url)
}

func IfExists(path string) *Pipe {
	p := NewPipe()
	p.Pipeline.SetExitOnError(true)
	return p.Pipe(std.IfExists(path)).Wait()
}

// ListFiles creates a pipeline with the file listing of path
func ListFiles(path string) *Pipe {
	return NewPipe().Pipe(std.ListFiles(path))
}

// Do creates a pipeline with a POST HTTP request
func Post(url string) *Pipe {
	return NewPipe().Post(url)
}

// Slice creates a pipeline with a new line for each slice item
func Slice(s []string) *Pipe {
	return Echo(strings.Join(s, "\n") + "\n")
}

// StdExec creates a pipeline with a standard system command
func StdExec(name string, arg ...string) *Pipe {
	return NewPipe().Pipeline.Exec(name, arg...)
}

// Stdin creates a pipeline with stdin as input
func Stdin() *Pipe {
	return NewPipe().Pipe(std.Stdin())
}

// Program shortcuts:

// AppendFile reads the input and appends it to the file path, creating it if necessary,
// and outputs the number of bytes successfully written
func (p *Pipe) AppendFile(path string) (int64, error) {
	return p.Pipe(std.AppendFile(path)).Int64()
}

// CountLines returns the number of lines of input, or an error.
func (p *Pipe) CountLines() (int, error) {
	return p.Pipe(std.CountLines()).Int()
}

// Get reads the input as the request body, sends the request and outputs the response
func (p *Pipe) Do(req *http.Request) *Pipe {
	return p.Pipe(std.Do(req, p.httpClient))
}

// Exec executes cmdLine using sh/shell, using input as stdin and outputs the result
func (p *Pipe) Exec(cmdLine string) *Pipe {
	return p.Pipe(shell.Exec(cmdLine))
}

// ExecForEach renders cmdLine as a Go template for each line of input, running
// the resulting command, and outputs the combined result of these commands in sequence
func (p *Pipe) ExecForEach(cmdLine string) *Pipe {
	return p.Pipe(shell.ExecForEach(cmdLine))
}

// Get reads the input as the request body, sends a GET request and outputs the response
func (p *Pipe) Get(url string) *Pipe {
	return p.Pipe(std.Get(url, p.httpClient))
}

// JQ reads the input (presumed to be JSON), executes the query and outputs the result
func (p *Pipe) JQ(query string) *Pipe {
	return p.Pipe(gojq.JQ(query))
}

// Get reads the input as the request body, sends a POST request and outputs the response
func (p *Pipe) Post(url string) *Pipe {
	return p.Pipe(std.Post(url, p.httpClient))
}

// SHA256Sum reads the input and outputs the hex-encoded SHA-256 hash
func (p *Pipe) SHA256Sum() (string, error) {
	return p.Pipe(std.SHA256Sum()).String()
}

// Tee reads the input and copies it to each of the supplied writers, like Unix tee(1)
func (p *Pipe) Tee(writers ...io.Writer) *Pipe {
	if len(writers) == 0 {
		p.Pipe(std.Tee(p.stdout)) // If no writers Tee with Pipe.stdout
	}
	return p.Pipe(std.Tee(writers...))
}

// WriteFile reads the input and writes it to the file path, truncating it if it exists,
// and outputs the number of bytes successfully written
func (p *Pipe) WriteFile(path string) (int64, error) {
	return p.Pipe(std.WriteFile(path)).Int64()
}

// With* functions:

// WithHTTPClient sets the HTTP client c for use with subsequent requests
func (p *Pipe) WithHTTPClient(c *http.Client) *Pipe {
	p.httpClient = c
	return p
}

// WithStdout sets the pipe's standard output to the writer w
func (p *Pipe) WithStdout(w io.Writer) *Pipe {
	p.stdout = w
	p.Pipeline.WithStdout(w)
	p.Pipeline.WithStderr(w)
	return p
}

func (p *Pipe) WithStderr(w io.Writer) *Pipe {
	p.Pipeline.WithStderr(w)
	p.Pipeline.SetCombinedOutput(false)
	return p
}

func NewReadAutoCloser(r io.Reader) io.Reader {
	return pipeline.NewReadOnlyPipe(r)
}
