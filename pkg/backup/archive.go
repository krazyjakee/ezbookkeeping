package backup

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

// archiveWriter writes a (optionally encrypted) tar.gz backup archive and records the size and checksum of each entry
type archiveWriter struct {
	file          *os.File
	bufferedFile  *bufio.Writer
	encryptWriter io.WriteCloser
	gzipWriter    *gzip.Writer
	tarWriter     *tar.Writer
	modTime       time.Time
	entries       []*ManifestEntry
	entryNames    map[string]bool
}

func newArchiveWriter(filePath string, passphrase string, encrypt bool, modTime time.Time) (*archiveWriter, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)

	if err != nil {
		return nil, err
	}

	writer := &archiveWriter{
		file:         file,
		bufferedFile: bufio.NewWriterSize(file, 256*1024),
		modTime:      modTime,
		entryNames:   make(map[string]bool),
	}

	var target io.Writer = writer.bufferedFile

	if encrypt {
		writer.encryptWriter, err = newEncryptingWriter(writer.bufferedFile, passphrase)

		if err != nil {
			_ = file.Close()
			return nil, err
		}

		target = writer.encryptWriter
	}

	writer.gzipWriter, err = gzip.NewWriterLevel(target, gzip.BestCompression)

	if err != nil {
		_ = file.Close()
		return nil, err
	}

	writer.tarWriter = tar.NewWriter(writer.gzipWriter)

	return writer, nil
}

func (w *archiveWriter) addEntry(name string, size int64, reader io.Reader) (*ManifestEntry, error) {
	if err := validateEntryName(name); err != nil {
		return nil, err
	}

	if w.entryNames[name] {
		return nil, fmt.Errorf("duplicate backup entry \"%s\"", name)
	}

	err := w.tarWriter.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     name,
		Size:     size,
		Mode:     0600,
		ModTime:  w.modTime,
		Format:   tar.FormatPAX,
	})

	if err != nil {
		return nil, err
	}

	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(w.tarWriter, hash), reader)

	if err != nil {
		return nil, err
	}

	if written != size {
		return nil, fmt.Errorf("backup entry \"%s\" size changed while writing (expected %d bytes, got %d bytes)", name, size, written)
	}

	entry := &ManifestEntry{
		Name:   name,
		Size:   size,
		SHA256: hex.EncodeToString(hash.Sum(nil)),
	}

	if name != manifestEntryName {
		w.entries = append(w.entries, entry)
	}

	w.entryNames[name] = true

	return entry, nil
}

func (w *archiveWriter) addBytes(name string, data []byte) (*ManifestEntry, error) {
	return w.addEntry(name, int64(len(data)), bytes.NewReader(data))
}

func (w *archiveWriter) addFile(name string, filePath string) (*ManifestEntry, error) {
	file, err := os.Open(filePath)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	fileInfo, err := file.Stat()

	if err != nil {
		return nil, err
	}

	return w.addEntry(name, fileInfo.Size(), file)
}

// finish writes the manifest as the last entry and flushes all data to disk
func (w *archiveWriter) finish(manifest *Manifest) error {
	manifest.Entries = w.entries
	manifestData, err := json.MarshalIndent(manifest, "", "  ")

	if err != nil {
		return err
	}

	if _, err := w.addBytes(manifestEntryName, manifestData); err != nil {
		return err
	}

	if err := w.tarWriter.Close(); err != nil {
		return err
	}

	if err := w.gzipWriter.Close(); err != nil {
		return err
	}

	if w.encryptWriter != nil {
		if err := w.encryptWriter.Close(); err != nil {
			return err
		}
	}

	if err := w.bufferedFile.Flush(); err != nil {
		return err
	}

	if err := w.file.Sync(); err != nil {
		return err
	}

	return w.file.Close()
}

// abort closes the underlying file without finishing the archive
func (w *archiveWriter) abort() {
	_ = w.file.Close()
}

// archiveEntryHandler handles an archive entry, the reader must not be used after the handler returns
type archiveEntryHandler func(name string, size int64, reader io.Reader) error

// archiveReadResult represents the result of reading a whole backup archive
type archiveReadResult struct {
	Manifest  *Manifest
	Entries   map[string]*ManifestEntry
	Encrypted bool
}

// readArchive reads all the entries of the backup archive, computes their checksums, and parses the manifest.
// The handler (optional) is called for every entry except the manifest.
// It returns an error if the archive is corrupted, truncated or cannot be decrypted, it does NOT compare the computed checksums with the manifest.
func readArchive(filePath string, passphrase string, handler archiveEntryHandler) (*archiveReadResult, error) {
	encrypted, err := IsEncryptedBackupFile(filePath)

	if err != nil {
		return nil, err
	}

	file, err := os.Open(filePath)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	var reader io.Reader = bufio.NewReaderSize(file, 256*1024)

	if encrypted {
		if passphrase == "" {
			return nil, ErrBackupPassphraseRequired
		}

		reader, err = newDecryptingReader(reader, passphrase)

		if err != nil {
			return nil, err
		}
	}

	gzipReader, err := gzip.NewReader(reader)

	if err != nil {
		return nil, wrapArchiveReadError(err)
	}

	defer gzipReader.Close()

	gzipReader.Multistream(false)
	tarReader := tar.NewReader(gzipReader)

	result := &archiveReadResult{
		Entries:   make(map[string]*ManifestEntry),
		Encrypted: encrypted,
	}

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		} else if err != nil {
			return nil, wrapArchiveReadError(err)
		}

		if header.Typeflag != tar.TypeReg {
			return nil, fmt.Errorf("%w: unexpected entry type of \"%s\"", ErrBackupCorrupted, header.Name)
		}

		if err := validateEntryName(header.Name); err != nil {
			return nil, err
		}

		if _, exists := result.Entries[header.Name]; exists || (header.Name == manifestEntryName && result.Manifest != nil) {
			return nil, fmt.Errorf("%w: duplicate entry \"%s\"", ErrBackupCorrupted, header.Name)
		}

		if result.Manifest != nil {
			return nil, fmt.Errorf("%w: unexpected entry \"%s\" after manifest", ErrBackupCorrupted, header.Name)
		}

		if header.Name == manifestEntryName {
			if header.Size > maxManifestSize {
				return nil, fmt.Errorf("%w: manifest is too large", ErrBackupCorrupted)
			}

			manifestData, err := io.ReadAll(tarReader)

			if err != nil {
				return nil, wrapArchiveReadError(err)
			}

			manifest := &Manifest{}

			if err := json.Unmarshal(manifestData, manifest); err != nil {
				return nil, fmt.Errorf("%w: invalid manifest (%s)", ErrBackupCorrupted, err.Error())
			}

			result.Manifest = manifest
			continue
		}

		hash := sha256.New()
		countingReader := &countingReader{reader: io.TeeReader(tarReader, hash)}

		if handler != nil {
			if err := handler(header.Name, header.Size, countingReader); err != nil {
				return nil, wrapArchiveReadError(err)
			}
		}

		if _, err := io.Copy(io.Discard, countingReader); err != nil {
			return nil, wrapArchiveReadError(err)
		}

		result.Entries[header.Name] = &ManifestEntry{
			Name:   header.Name,
			Size:   countingReader.count,
			SHA256: hex.EncodeToString(hash.Sum(nil)),
		}
	}

	// read all the remaining data, so that the gzip checksum is verified and the encrypted final chunk is checked
	if _, err := io.Copy(io.Discard, gzipReader); err != nil {
		return nil, wrapArchiveReadError(err)
	}

	if trailingSize, err := io.Copy(io.Discard, reader); err != nil {
		return nil, wrapArchiveReadError(err)
	} else if trailingSize > 0 {
		return nil, fmt.Errorf("%w: unexpected data after the end of archive", ErrBackupCorrupted)
	}

	if result.Manifest == nil {
		return nil, ErrBackupManifestMissing
	}

	return result, nil
}

func wrapArchiveReadError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, ErrBackupCorrupted) || errors.Is(err, ErrBackupTruncated) || errors.Is(err, ErrBackupWrongPassphrase) ||
		errors.Is(err, ErrBackupIncompatible) || errors.Is(err, ErrBackupManifestMissing) {
		return err
	}

	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: %s", ErrBackupTruncated, err.Error())
	}

	var flateCorruptInputError flate.CorruptInputError

	if errors.As(err, &flateCorruptInputError) {
		return fmt.Errorf("%w: %s", ErrBackupCorrupted, err.Error())
	}

	if errors.Is(err, gzip.ErrChecksum) || errors.Is(err, gzip.ErrHeader) || errors.Is(err, tar.ErrHeader) {
		return fmt.Errorf("%w: %s", ErrBackupCorrupted, err.Error())
	}

	return err
}

type countingReader struct {
	reader io.Reader
	count  int64
}

func (r *countingReader) Read(data []byte) (int, error) {
	size, err := r.reader.Read(data)
	r.count += int64(size)
	return size, err
}
