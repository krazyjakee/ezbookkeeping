package donut

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// ErrPdfToolsNotFound is returned when the poppler tools (pdftoppm and pdftotext) are not installed
var ErrPdfToolsNotFound = errors.New("PDF documents cannot be read because pdftoppm / pdftotext (poppler-utils) are not installed")

// pdfToolTimeout is the maximum time each poppler tool may run
const pdfToolTimeout = time.Minute

// ReadPDF returns the first page of the PDF document rendered as png (144 dpi) and the text layer of its first pages (empty for scanned documents).
// It runs the poppler tools in separate processes, so a malformed document cannot affect the server process
func ReadPDF(data []byte, maxTextPages int) ([]byte, string, error) {
	pdftoppm, err := exec.LookPath("pdftoppm")

	if err != nil {
		return nil, "", ErrPdfToolsNotFound
	}

	pdftotext, err := exec.LookPath("pdftotext")

	if err != nil {
		return nil, "", ErrPdfToolsNotFound
	}

	dir, err := os.MkdirTemp("", "ezbookkeeping-donut-pdf-")

	if err != nil {
		return nil, "", err
	}

	defer os.RemoveAll(dir)

	input := filepath.Join(dir, "document.pdf")

	if err := os.WriteFile(input, data, 0o600); err != nil {
		return nil, "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), pdfToolTimeout)
	defer cancel()

	var stderr bytes.Buffer
	render := exec.CommandContext(ctx, pdftoppm, "-png", "-r", "144", "-f", "1", "-l", "1", "-singlefile", input, filepath.Join(dir, "page"))
	render.Stderr = &stderr

	if err := render.Run(); err != nil {
		return nil, "", fmt.Errorf("cannot render the PDF document: %s %s", err.Error(), stderr.String())
	}

	page, err := os.ReadFile(filepath.Join(dir, "page.png"))

	if err != nil {
		return nil, "", err
	}

	var text bytes.Buffer
	extract := exec.CommandContext(ctx, pdftotext, "-layout", "-enc", "UTF-8", "-f", "1", "-l", strconv.Itoa(max(maxTextPages, 1)), input, "-")
	extract.Stdout = &text

	// a document without a text layer is not an error, the rendered page is still read
	if err := extract.Run(); err != nil {
		text.Reset()
	}

	return page, text.String(), nil
}
