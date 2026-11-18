package donut

import (
	"bufio"
	"encoding/json/v2"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const fakeWorkerEnvName = "EBK_DONUT_FAKE_WORKER"

// TestMain lets the test binary act as a fake worker process when started by the worker manager in the tests below
func TestMain(m *testing.M) {
	if mode := os.Getenv(fakeWorkerEnvName); mode != "" {
		runFakeWorker(mode)
		os.Exit(0)
	}

	os.Exit(m.Run())
}

func runFakeWorker(mode string) {
	if mode == "fail-to-load" {
		fmt.Fprintln(os.Stderr, "cannot load libonnxruntime.so.1")
		fmt.Println(`{"id":0,"error":"cannot load ONNX Runtime"}`)
		return
	}

	fmt.Println(`{"id":0,"ready":true}`)
	input := bufio.NewScanner(os.Stdin)
	input.Buffer(make([]byte, 1024*1024), 16*1024*1024)

	for input.Scan() {
		var request WorkerRequest
		json.Unmarshal(input.Bytes(), &request)

		switch {
		case mode == "crash":
			os.Exit(3)
		case mode == "hang":
			time.Sleep(time.Hour)
		}

		args := strings.Join(os.Args[1:], " ")
		response, _ := json.Marshal(&WorkerResponse{ID: request.ID, Results: []*WorkerModelResult{
			{Model: "echo", Sequence: "<s_total>" + string(request.Image) + "</s_total><s_args>" + args + "</s_args>", Tokens: 3},
		}})
		fmt.Println(string(response))
	}
}

func newFakeWorker(t *testing.T, mode string, idle time.Duration) *Worker {
	t.Setenv(fakeWorkerEnvName, mode)
	executable, err := os.Executable()
	assert.Nil(t, err)
	worker := NewWorker(executable, "libonnxruntime.so.1", 2, mode == "low-memory", idle, nil)
	t.Cleanup(worker.Stop)
	return worker
}

func TestWorker_ReadsAndReusesProcess(t *testing.T) {
	worker := newFakeWorker(t, "echo", time.Minute)

	results, err := worker.Read([]string{"/models/a", "/models/b"}, []byte("12.34"), 5*time.Second)
	assert.Nil(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, map[string]any{"total": "12.34", "args": "-library libonnxruntime.so.1 -threads 2 /models/a /models/b"}, TokensToJSON(results[0].Sequence))

	pid := worker.process.cmd.Process.Pid
	results, err = worker.Read([]string{"/models/a"}, []byte("5"), 5*time.Second)
	assert.Nil(t, err)
	assert.Equal(t, "5", TokensToJSON(results[0].Sequence)["total"])
	assert.Equal(t, pid, worker.process.cmd.Process.Pid)
}

func TestWorker_PassesLowMemoryFlag(t *testing.T) {
	worker := newFakeWorker(t, "low-memory", time.Minute)

	results, err := worker.Read([]string{"/models/a"}, []byte("1"), 5*time.Second)
	assert.Nil(t, err)
	assert.Equal(t, "-library libonnxruntime.so.1 -threads 2 -low-memory /models/a", TokensToJSON(results[0].Sequence)["args"])
}

func TestWorker_StopsWhenIdleAndRestarts(t *testing.T) {
	worker := newFakeWorker(t, "echo", 50*time.Millisecond)

	_, err := worker.Read(nil, []byte("1"), 5*time.Second)
	assert.Nil(t, err)

	time.Sleep(300 * time.Millisecond)
	worker.mutex.Lock()
	assert.Nil(t, worker.process)
	worker.mutex.Unlock()

	results, err := worker.Read(nil, []byte("2"), 5*time.Second)
	assert.Nil(t, err)
	assert.Equal(t, "2", TokensToJSON(results[0].Sequence)["total"])
}

func TestWorker_ReportsLoadFailure(t *testing.T) {
	worker := newFakeWorker(t, "fail-to-load", time.Minute)

	_, err := worker.Read(nil, []byte("1"), 5*time.Second)
	assert.ErrorIs(t, err, ErrWorkerUnavailable)
	assert.Contains(t, err.Error(), "cannot load ONNX Runtime")
}

func TestWorker_RecoversFromCrashAndTimeout(t *testing.T) {
	worker := newFakeWorker(t, "crash", time.Minute)

	_, err := worker.Read(nil, []byte("1"), 5*time.Second)
	assert.ErrorIs(t, err, ErrWorkerUnavailable)
	assert.Nil(t, worker.process)

	t.Setenv(fakeWorkerEnvName, "hang")
	started := time.Now()
	_, err = worker.Read(nil, []byte("1"), 200*time.Millisecond)
	assert.ErrorIs(t, err, ErrWorkerUnavailable)
	assert.Contains(t, err.Error(), "no response")
	assert.Less(t, time.Since(started), 3*time.Second)
	assert.Nil(t, worker.process)

	t.Setenv(fakeWorkerEnvName, "echo")
	results, err := worker.Read(nil, []byte("7"), 5*time.Second)
	assert.Nil(t, err)
	assert.Equal(t, "7", TokensToJSON(results[0].Sequence)["total"])
}

func TestWorker_MissingBinary(t *testing.T) {
	worker := NewWorker("/nonexistent/ezbookkeeping-donut", "lib", 1, false, time.Minute, nil)
	_, err := worker.Read(nil, []byte("1"), time.Second)
	assert.ErrorIs(t, err, ErrWorkerUnavailable)
}
