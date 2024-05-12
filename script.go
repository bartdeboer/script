package script

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/bitfield/script/gojq"
	"github.com/bitfield/script/pipeline"
	"github.com/bitfield/script/shell"
	"github.com/bitfield/script/std"
)

type Pipe struct {
	*pipeline.Pipeline
	stdout io.Writer

	httpClient *http.Client
}

func NewPipe() *Pipe {
	return &Pipe{
		pipeline.NewPipeline(),
		os.Stdout,
		http.DefaultClient,
	}
}

func (p *Pipe) PipeE(prog pipeline.Process) *Pipe {
	p.Pipeline.PipeE(prog)
	return p
}

func (p *Pipe) Pipe(prog pipeline.ProcessE) *Pipe {
	p.Pipeline.Pipe(prog)
	return p
}

// For backwards compatibility
func (p *Pipe) Filter(prog pipeline.Process) *Pipe {
	return p.PipeE(prog)
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

func (p *Pipe) Wait() *Pipe {
	p.Pipeline.Wait()
	return p
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
	return NewPipe().PipeE(std.FindFiles(dir))
}

// Do creates a pipeline with a GET HTTP request
func Get(url string) *Pipe {
	return NewPipe().Get(url)
}

// IfExists creates a pipeline if the file exists
func IfExists(path string) *Pipe {
	p := NewPipe()
	p.Pipeline.SetExitOnError(true)
	return p.PipeE(std.IfExists(path)).Wait()
}

// ListFiles creates a pipeline with the file listing of path
func ListFiles(path string) *Pipe {
	return NewPipe().PipeE(std.ListFiles(path))
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
	return NewPipe().StdExec(name, arg...)
}

// Stdin creates a pipeline with stdin as input
func Stdin() *Pipe {
	return NewPipe().PipeE(std.Stdin())
}

// Program shortcuts:

// AppendFile reads the input and appends it to the file path, creating it if necessary,
// and outputs the number of bytes successfully written
func (p *Pipe) AppendFile(path string) (int64, error) {
	return p.PipeE(std.AppendFile(path)).Int64()
}

// Basename reads each line as a file path and outputs each path with any leading directory components removed
func (p *Pipe) Basename() *Pipe {
	return p.PipeE(std.Basename())
}

// Column reads each line and outputs column col, where columns are whitespace delimited and the first column is column 1
func (p *Pipe) Column(col int) *Pipe {
	return p.PipeE(std.Column(col))
}

// Concat reads each line as a file path and outputs the file contents
func (p *Pipe) Concat() *Pipe {
	return p.PipeE(std.Concat())
}

// CountLines returns the number of lines of input, or an error.
func (p *Pipe) CountLines() (int, error) {
	return p.PipeE(std.CountLines()).Int()
}

// Dirname reads each line as a file path and outputs each path with just the leading directory remaining
func (p *Pipe) Dirname() *Pipe {
	return p.PipeE(std.Dirname())
}

// Get reads the input as the request body, sends the request and outputs the response
func (p *Pipe) Do(req *http.Request) *Pipe {
	return p.PipeE(std.Do(req, p.httpClient))
}

// Deprecated: use [Pipe.FilterLine] or [Pipe.FilterScan] instead
func (p *Pipe) EachLine(process func(string, *strings.Builder)) *Pipe {
	return p.PipeE(std.EachLine(process))
}

// Echo ignores its input and outputs string s
func (p *Pipe) Echo(s string) *Pipe {
	return p.PipeE(std.Echo(s))
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

// FilterLine reads the input, calls the function filter on each line and outputs the result
func (p *Pipe) FilterLine(filter func(string) string) *Pipe {
	return p.PipeE(std.FilterLine(filter))
}

// First produces only the first n lines of the input and outputs only the first n number of lines
func (p *Pipe) First(n int) *Pipe {
	return p.PipeE(std.First(n))
}

// Freq reads the input and outputs only the unique lines, each prefixed with
// a frequency count, in descending numerical order
func (p *Pipe) Freq() *Pipe {
	return p.PipeE(std.Freq())
}

// Get reads the input as the request body, sends a GET request and outputs the response
func (p *Pipe) Get(url string) *Pipe {
	return p.PipeE(std.Get(url, p.httpClient))
}

// Join reads all the lines and joins them into a single space-separated string
func (p *Pipe) Join() *Pipe {
	return p.PipeE(std.Join())
}

// JQ reads the input (presumed to be JSON), executes the query and outputs the result
func (p *Pipe) JQ(query string) *Pipe {
	return p.PipeE(gojq.JQ(query))
}

// First reads the input and outputs only the last n number of lines
func (p *Pipe) Last(n int) *Pipe {
	return p.PipeE(std.Last(n))
}

// Match reads the input and outputs lines that contain the string s
func (p *Pipe) Match(s string) *Pipe {
	return p.PipeE(std.Match(s))
}

// MatchRegexp reads the input and outputs lines that match the compiled regexp re
func (p *Pipe) MatchRegexp(re *regexp.Regexp) *Pipe {
	return p.PipeE(std.MatchRegexp(re))
}

// Get reads the input as the request body, sends a POST request and outputs the response
func (p *Pipe) Post(url string) *Pipe {
	return p.PipeE(std.Post(url, p.httpClient))
}

// Reject reads the input and outputs lines that do not contain the string s
func (p *Pipe) Reject(s string) *Pipe {
	return p.PipeE(std.Reject(s))
}

// RejectRegexp reads the input and outputs lines that do not match the compiled regexp re
func (p *Pipe) RejectRegexp(re *regexp.Regexp) *Pipe {
	return p.PipeE(std.RejectRegexp(re))
}

// Replace reads the input and replaces all occurrences of the string search with the string replace
func (p *Pipe) Replace(search, replace string) *Pipe {
	return p.PipeE(std.Replace(search, replace))
}

// ReplaceRegexp reads the input and replaces all matches of the compiled regexp re with the string replace
func (p *Pipe) ReplaceRegexp(re *regexp.Regexp, replace string) *Pipe {
	return p.PipeE(std.ReplaceRegexp(re, replace))
}

// Scanner reads the input into a scanner, calls the function filter on each line and outputs the result
func (p *Pipe) Scanner(filter func(string, io.Writer)) *Pipe {
	return p.PipeE(std.Scanner(filter))
}

// SHA256Sum reads the input and outputs the hex-encoded SHA-256 hash
func (p *Pipe) SHA256Sum() (string, error) {
	return p.PipeE(std.SHA256Sum()).String()
}

// SHA256Sum reads the input and outputs the hex-encoded SHA-256 hash of each line
func (p *Pipe) SHA256Sums() *Pipe {
	return p.PipeE(std.SHA256Sums())
}

// StdExec executes the command with name and arguments, using input as stdin and outputs the result
func (p *Pipe) StdExec(name string, arg ...string) *Pipe {
	return p.Pipe(std.Exec(name, arg...))
}

// Tee reads the input and copies it to each of the supplied writers, like Unix tee(1)
func (p *Pipe) Tee(writers ...io.Writer) *Pipe {
	if len(writers) == 0 {
		p.PipeE(std.Tee(p.stdout)) // If no writers Tee with Pipe.stdout
	}
	return p.PipeE(std.Tee(writers...))
}

// WriteFile reads the input and writes it to the file path, truncating it if it exists,
// and outputs the number of bytes successfully written
func (p *Pipe) WriteFile(path string) (int64, error) {
	return p.PipeE(std.WriteFile(path)).Int64()
}

// With* functions:

// WithError sets the error err on the pipe
func (p *Pipe) WithError(err error) *Pipe {
	p.Pipeline = p.Pipeline.WithError(err)
	return p
}

// WithHTTPClient sets the HTTP client c for use with subsequent requests
func (p *Pipe) WithHTTPClient(c *http.Client) *Pipe {
	p.httpClient = c
	return p
}

// WithReader sets the pipe's input reader to r
func (p *Pipe) WithReader(r io.Reader) *Pipe {
	p.Pipeline = p.Pipeline.WithReader(r)
	return p
}

// WithStderr redirects the standard error output for commands
func (p *Pipe) WithStderr(w io.Writer) *Pipe {
	p.Pipeline = p.Pipeline.WithStderr(w)
	return p
}

// WithStdout sets the pipe's standard output to the writer w
func (p *Pipe) WithStdout(w io.Writer) *Pipe {
	p.stdout = w
	p.Pipeline = p.Pipeline.WithStdout(w)
	return p
}
