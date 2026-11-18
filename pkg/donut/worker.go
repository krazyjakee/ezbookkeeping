package donut

import (
	"bufio"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ErrWorkerUnavailable is returned when the worker process cannot be started or stops unexpectedly
var ErrWorkerUnavailable = errors.New("the receipt recognition worker is not available")

// workerStartTimeout is the maximum time to wait until the worker has loaded the models
const workerStartTimeout = 3 * time.Minute

// Worker manages the worker process which runs the models. The process is started on first use, serves one request at a time,
// and is stopped after it has been idle for the idle timeout, so the memory of the models is only used while documents are read
type Worker struct {
	path        string
	library     string
	threads     int
	lowMemory   bool
	idleTimeout time.Duration
	logf        func(format string, args ...any)

	mutex     sync.Mutex
	process   *workerProcess
	idleTimer *time.Timer
	nextID    int64
}

type workerProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	lines  chan []byte
	stderr *tailBuffer
}

// NewWorker returns a worker manager for the worker binary at path
func NewWorker(path string, library string, threads int, lowMemory bool, idleTimeout time.Duration, logf func(format string, args ...any)) *Worker {
	if logf == nil {
		logf = func(string, ...any) {}
	}

	return &Worker{path: path, library: library, threads: threads, lowMemory: lowMemory, idleTimeout: idleTimeout, logf: logf}
}

// Read sends the image to the worker and returns the output of every model in modelDirs
func (w *Worker) Read(modelDirs []string, image []byte, timeout time.Duration) ([]*WorkerModelResult, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.idleTimer != nil {
		w.idleTimer.Stop()
	}

	if w.process == nil {
		process, err := w.start(modelDirs)

		if err != nil {
			return nil, err
		}

		w.process = process
	}

	w.nextID++
	request, err := json.Marshal(&WorkerRequest{ID: w.nextID, Image: image})

	if err != nil {
		return nil, err
	}

	if _, err := w.process.stdin.Write(append(request, '\n')); err != nil {
		w.stopLocked()
		return nil, fmt.Errorf("%w: %s", ErrWorkerUnavailable, err.Error())
	}

	response, err := w.process.receive(timeout)

	if err != nil {
		w.stopLocked()
		return nil, err
	}

	w.idleTimer = time.AfterFunc(w.idleTimeout, w.stopIdle)

	if response.ID != w.nextID {
		w.stopLocked()
		return nil, fmt.Errorf("%w: unexpected response", ErrWorkerUnavailable)
	}

	if response.Error != "" {
		return nil, errors.New(response.Error)
	}

	return response.Results, nil
}

func (w *Worker) start(modelDirs []string) (*workerProcess, error) {
	args := []string{"-library", w.library, "-threads", strconv.Itoa(w.threads)}

	if w.lowMemory {
		args = append(args, "-low-memory")
	}

	cmd := exec.Command(w.path, append(args, modelDirs...)...)
	stdin, err := cmd.StdinPipe()

	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		return nil, err
	}

	process := &workerProcess{cmd: cmd, stdin: stdin, lines: make(chan []byte, 1), stderr: &tailBuffer{limit: 4096}}
	cmd.Stderr = process.stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("%w: cannot start %s: %s", ErrWorkerUnavailable, w.path, err.Error())
	}

	w.logf("started the donut worker (pid %d)", cmd.Process.Pid)

	go func() {
		reader := bufio.NewReaderSize(stdout, 64*1024)

		for {
			line, err := reader.ReadBytes('\n')

			if len(line) > 0 {
				process.lines <- line
			}

			if err != nil {
				close(process.lines)
				cmd.Wait()
				return
			}
		}
	}()

	ready, err := process.receive(workerStartTimeout)

	if err == nil && !ready.Ready {
		err = fmt.Errorf("%w: %s", ErrWorkerUnavailable, ready.Error)
	}

	if err != nil {
		process.kill()
		return nil, err
	}

	return process, nil
}

// receive waits for the next response of the worker process
func (p *workerProcess) receive(timeout time.Duration) (*WorkerResponse, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case line, ok := <-p.lines:
		if !ok {
			return nil, fmt.Errorf("%w: the worker stopped (%s)", ErrWorkerUnavailable, strings.TrimSpace(p.stderr.String()))
		}

		response := &WorkerResponse{}

		if err := json.Unmarshal(line, response); err != nil {
			return nil, fmt.Errorf("%w: invalid response", ErrWorkerUnavailable)
		}

		return response, nil
	case <-timer.C:
		return nil, fmt.Errorf("%w: no response within %s", ErrWorkerUnavailable, timeout)
	}
}

func (p *workerProcess) kill() {
	p.stdin.Close()

	if p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}

	// drain the output so the reader goroutine can exit
	go func() {
		for range p.lines {
		}
	}()
}

func (w *Worker) stopIdle() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.process != nil {
		w.logf("stopping the idle donut worker")
		w.stopLocked()
	}
}

func (w *Worker) stopLocked() {
	if w.process != nil {
		w.process.kill()
		w.process = nil
	}
}

// Stop stops the worker process if it is running
func (w *Worker) Stop() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.idleTimer != nil {
		w.idleTimer.Stop()
	}

	w.stopLocked()
}

// tailBuffer keeps the last bytes written to it (the end of the stderr output of the worker, for error messages)
type tailBuffer struct {
	mutex sync.Mutex
	limit int
	data  []byte
}

func (b *tailBuffer) Write(data []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.data = append(b.data, data...)

	if len(b.data) > b.limit {
		b.data = b.data[len(b.data)-b.limit:]
	}

	return len(data), nil
}

func (b *tailBuffer) String() string {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return string(b.data)
}
