package ui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

// Console wraps interactive CLI input and output.
type Console struct {
	in     io.Reader
	reader *bufio.Reader
	out    io.Writer
	err    io.Writer
}

func NewConsole(in io.Reader, out, err io.Writer) *Console {
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	if err == nil {
		err = os.Stderr
	}
	return &Console{
		in:     in,
		reader: bufio.NewReader(in),
		out:    out,
		err:    err,
	}
}

func DefaultConsole() *Console {
	return NewConsole(os.Stdin, os.Stdout, os.Stderr)
}

func (c *Console) IsInteractive() bool {
	f, ok := c.in.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

func (c *Console) Printf(format string, args ...any) {
	fmt.Fprintf(c.out, format, args...)
}

func (c *Console) Println(args ...any) {
	fmt.Fprintln(c.out, args...)
}

func (c *Console) PrintErrln(args ...any) {
	fmt.Fprintln(c.err, args...)
}

func (c *Console) PrintStep(step, total int, title string) {
	c.Printf("\n[%d/%d] %s\n", step, total, title)
}

func (c *Console) PrintSuccess(msg string) {
	c.Printf("✓ %s\n", msg)
}

func (c *Console) PrintError(msg string) {
	c.PrintErrln("✗", msg)
}

func (c *Console) ReadLine(prompt string) (string, error) {
	if prompt != "" {
		c.Printf("%s", prompt)
	}
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// Select prints a numbered menu and returns the chosen options index (0-based).
func (c *Console) Select(prompt string, options []string, defaultIndex int) (int, error) {
	if len(options) == 0 {
		return 0, errors.New("options cannot be empty")
	}
	c.Println(prompt)
	for i, opt := range options {
		c.Printf("  %d) %s\n", i+1, opt)
	}
	defaultDisplay := strconv.Itoa(defaultIndex + 1)
	line, err := c.ReadLine(fmt.Sprintf("Enter number [%s]: ", defaultDisplay))
	if err != nil {
		return 0, err
	}
	if line == "" {
		return defaultIndex, nil
	}
	n, err := strconv.Atoi(line)
	if err != nil || n < 1 || n > len(options) {
		return 0, fmt.Errorf("invalid option: %s", line)
	}
	return n - 1, nil
}

// Confirm asks Y/n. When defaultYes is true, Enter means yes.
func (c *Console) Confirm(prompt string, defaultYes bool) (bool, error) {
	hint := "[y/N]"
	if defaultYes {
		hint = "[Y/n]"
	}
	line, err := c.ReadLine(fmt.Sprintf("%s %s: ", prompt, hint))
	if err != nil {
		return false, err
	}
	line = strings.ToLower(strings.TrimSpace(line))
	if line == "" {
		return defaultYes, nil
	}
	return line == "y" || line == "yes", nil
}

// Input reads one line. Enter keeps defaultVal when it is non-empty.
func (c *Console) Input(prompt, defaultVal string) (string, error) {
	hint := prompt
	if defaultVal != "" {
		hint = fmt.Sprintf("%s [%s]: ", prompt, defaultVal)
	} else {
		hint = prompt + ": "
	}
	line, err := c.ReadLine(hint)
	if err != nil {
		return "", err
	}
	if line == "" {
		return defaultVal, nil
	}
	return line, nil
}

// WaitSpinner shows a spinner while fn runs.
func (c *Console) WaitSpinner(message string, fn func() error) error {
	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	frames := []string{"|", "/", "-", "\\"}
	i := 0
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			c.Printf("\r%s\r", strings.Repeat(" ", len(message)+4))
			return err
		case <-ticker.C:
			c.Printf("\r%s %s ", message, frames[i%len(frames)])
			i++
		}
	}
}

// OpenBrowser opens url in the system browser.
func (c *Console) OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Stdout = c.out
	cmd.Stderr = c.err
	return cmd.Run()
}
