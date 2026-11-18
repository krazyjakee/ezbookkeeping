package donut

// The worker process reads one json request per line from stdin and writes one json response per line to stdout

// WorkerRequest asks the worker to read an image with all loaded models
type WorkerRequest struct {
	ID    int64  `json:"id"`
	Image []byte `json:"image"`
}

// WorkerModelResult is the output of one model for an image
type WorkerModelResult struct {
	Model    string `json:"model"`
	Sequence string `json:"sequence"`
	Tokens   int    `json:"tokens"`
	Millis   int64  `json:"millis"`
}

// WorkerResponse is the response of the worker to a request (the first response, with id 0, reports that the models are loaded)
type WorkerResponse struct {
	ID      int64                `json:"id"`
	Ready   bool                 `json:"ready,omitempty"`
	Results []*WorkerModelResult `json:"results,omitempty"`
	Error   string               `json:"error,omitempty"`
}
