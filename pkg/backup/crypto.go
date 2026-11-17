package backup

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/scrypt"
)

// Encrypted backup file layout (all integers are big-endian):
//
//	header:  magic "EZBKENC1" | version (1 byte) | kdf (1 byte, 1 = scrypt) | scrypt log2(N) (1 byte) | scrypt r (1 byte) | scrypt p (1 byte)
//	         | plaintext chunk size (uint32) | salt (16 bytes) | nonce prefix (12 bytes) | key check tag (16 bytes)
//	records: final flag (1 byte, 0 = more chunks follow, 1 = last chunk) | ciphertext length (uint32) | AES-256-GCM ciphertext
//
// Each chunk is sealed with nonce = nonce prefix XOR chunk counter, and additional data = header | chunk counter | final flag,
// so that reordering, truncation (no final chunk) and appending (data after the final chunk) are all detected.
const (
	encryptionMagic          = "EZBKENC1"
	encryptionFormatVersion  = 1
	encryptionKdfScrypt      = 1
	encryptionSaltSize       = 16
	encryptionNonceSize      = 12
	encryptionTagSize        = 16
	encryptionKeySize        = 32
	encryptionHeaderBodySize = 1 + 1 + 1 + 1 + 1 + 4 + encryptionSaltSize + encryptionNonceSize
	encryptionHeaderSize     = len(encryptionMagic) + encryptionHeaderBodySize + encryptionTagSize

	defaultEncryptionChunkSize = 64 * 1024
	maxEncryptionChunkSize     = 16 * 1024 * 1024
	defaultScryptLogN          = 15
	defaultScryptR             = 8
	defaultScryptP             = 1
	maxScryptLogN              = 22

	keyCheckCounter = ^uint64(0)
)

var keyCheckAdditionalData = []byte("ezrbookkeeping-backup-key-check")

// IsEncryptedBackupFile returns whether the specified file is an encrypted backup file
func IsEncryptedBackupFile(filePath string) (bool, error) {
	file, err := os.Open(filePath)

	if err != nil {
		return false, err
	}

	defer file.Close()

	magic := make([]byte, len(encryptionMagic))
	_, err = io.ReadFull(file, magic)

	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return string(magic) == encryptionMagic, nil
}

type encryptionParams struct {
	logN        uint8
	r           uint8
	p           uint8
	chunkSize   uint32
	salt        []byte
	noncePrefix []byte
}

func (p *encryptionParams) headerWithoutKeyCheck() []byte {
	header := make([]byte, 0, len(encryptionMagic)+encryptionHeaderBodySize)
	header = append(header, encryptionMagic...)
	header = append(header, encryptionFormatVersion, encryptionKdfScrypt, p.logN, p.r, p.p)
	header = binary.BigEndian.AppendUint32(header, p.chunkSize)
	header = append(header, p.salt...)
	header = append(header, p.noncePrefix...)
	return header
}

func (p *encryptionParams) newAEAD(passphrase string) (cipher.AEAD, error) {
	if passphrase == "" {
		return nil, ErrBackupPassphraseRequired
	}

	key, err := scrypt.Key([]byte(passphrase), p.salt, 1<<p.logN, int(p.r), int(p.p), encryptionKeySize)

	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)

	if err != nil {
		return nil, err
	}

	return cipher.NewGCM(block)
}

func (p *encryptionParams) nonce(counter uint64) []byte {
	nonce := make([]byte, encryptionNonceSize)
	copy(nonce, p.noncePrefix)

	counterBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBytes, counter)

	for i := 0; i < 8; i++ {
		nonce[encryptionNonceSize-8+i] ^= counterBytes[i]
	}

	return nonce
}

func chunkAdditionalData(header []byte, counter uint64, final bool) []byte {
	additionalData := make([]byte, 0, len(header)+9)
	additionalData = append(additionalData, header...)
	additionalData = binary.BigEndian.AppendUint64(additionalData, counter)

	if final {
		additionalData = append(additionalData, 1)
	} else {
		additionalData = append(additionalData, 0)
	}

	return additionalData
}

// encryptingWriter encrypts the written data in chunked streaming AES-256-GCM mode
type encryptingWriter struct {
	writer  io.Writer
	aead    cipher.AEAD
	params  *encryptionParams
	header  []byte
	buffer  []byte
	counter uint64
	closed  bool
}

// newEncryptingWriter returns a writer which encrypts data with a key derived from the passphrase and writes them into the specified writer
func newEncryptingWriter(writer io.Writer, passphrase string) (io.WriteCloser, error) {
	return newEncryptingWriterWithParams(writer, passphrase, defaultScryptLogN, defaultEncryptionChunkSize)
}

func newEncryptingWriterWithParams(writer io.Writer, passphrase string, logN uint8, chunkSize uint32) (io.WriteCloser, error) {
	params := &encryptionParams{
		logN:        logN,
		r:           defaultScryptR,
		p:           defaultScryptP,
		chunkSize:   chunkSize,
		salt:        make([]byte, encryptionSaltSize),
		noncePrefix: make([]byte, encryptionNonceSize),
	}

	if _, err := rand.Read(params.salt); err != nil {
		return nil, err
	}

	if _, err := rand.Read(params.noncePrefix); err != nil {
		return nil, err
	}

	aead, err := params.newAEAD(passphrase)

	if err != nil {
		return nil, err
	}

	header := params.headerWithoutKeyCheck()
	keyCheck := aead.Seal(nil, params.nonce(keyCheckCounter), nil, append(append([]byte{}, header...), keyCheckAdditionalData...))

	if _, err := writer.Write(header); err != nil {
		return nil, err
	}

	if _, err := writer.Write(keyCheck); err != nil {
		return nil, err
	}

	return &encryptingWriter{
		writer: writer,
		aead:   aead,
		params: params,
		header: header,
		buffer: make([]byte, 0, chunkSize),
	}, nil
}

// Write encrypts and writes the data
func (w *encryptingWriter) Write(data []byte) (int, error) {
	if w.closed {
		return 0, errors.New("write to closed encrypting writer")
	}

	written := 0

	for len(data) > 0 {
		// only flush a full chunk when there is more data, so that the last chunk is always written as the final chunk
		if len(w.buffer) == int(w.params.chunkSize) {
			if err := w.flushChunk(false); err != nil {
				return written, err
			}
		}

		size := min(int(w.params.chunkSize)-len(w.buffer), len(data))
		w.buffer = append(w.buffer, data[:size]...)
		data = data[size:]
		written += size
	}

	return written, nil
}

// Close writes the final chunk, it does not close the underlying writer
func (w *encryptingWriter) Close() error {
	if w.closed {
		return nil
	}

	w.closed = true
	return w.flushChunk(true)
}

func (w *encryptingWriter) flushChunk(final bool) error {
	ciphertext := w.aead.Seal(nil, w.params.nonce(w.counter), w.buffer, chunkAdditionalData(w.header, w.counter, final))
	record := make([]byte, 0, 5)

	if final {
		record = append(record, 1)
	} else {
		record = append(record, 0)
	}

	record = binary.BigEndian.AppendUint32(record, uint32(len(ciphertext)))

	if _, err := w.writer.Write(record); err != nil {
		return err
	}

	if _, err := w.writer.Write(ciphertext); err != nil {
		return err
	}

	w.counter++
	w.buffer = w.buffer[:0]

	return nil
}

// decryptingReader decrypts the data which is encrypted by encryptingWriter
type decryptingReader struct {
	reader   *bufio.Reader
	aead     cipher.AEAD
	params   *encryptionParams
	header   []byte
	buffer   []byte
	counter  uint64
	finished bool
}

// newDecryptingReader returns a reader which reads and decrypts data from the specified reader
func newDecryptingReader(reader io.Reader, passphrase string) (io.Reader, error) {
	bufferedReader := bufio.NewReaderSize(reader, 64*1024)
	header := make([]byte, len(encryptionMagic)+encryptionHeaderBodySize)

	if _, err := io.ReadFull(bufferedReader, header); err != nil {
		return nil, fmt.Errorf("%w: cannot read encryption header", ErrBackupTruncated)
	}

	if string(header[:len(encryptionMagic)]) != encryptionMagic {
		return nil, fmt.Errorf("%w: not an encrypted backup file", ErrBackupCorrupted)
	}

	body := header[len(encryptionMagic):]

	if body[0] != encryptionFormatVersion || body[1] != encryptionKdfScrypt {
		return nil, fmt.Errorf("%w: unsupported encryption format", ErrBackupIncompatible)
	}

	params := &encryptionParams{
		logN:        body[2],
		r:           body[3],
		p:           body[4],
		chunkSize:   binary.BigEndian.Uint32(body[5:9]),
		salt:        append([]byte{}, body[9:9+encryptionSaltSize]...),
		noncePrefix: append([]byte{}, body[9+encryptionSaltSize:9+encryptionSaltSize+encryptionNonceSize]...),
	}

	if params.logN < 1 || params.logN > maxScryptLogN || params.r < 1 || params.r > 32 || params.p < 1 || params.p > 16 || params.chunkSize < 1 || params.chunkSize > maxEncryptionChunkSize {
		return nil, fmt.Errorf("%w: invalid encryption parameters", ErrBackupCorrupted)
	}

	keyCheck := make([]byte, encryptionTagSize)

	if _, err := io.ReadFull(bufferedReader, keyCheck); err != nil {
		return nil, fmt.Errorf("%w: cannot read encryption header", ErrBackupTruncated)
	}

	aead, err := params.newAEAD(passphrase)

	if err != nil {
		return nil, err
	}

	if _, err := aead.Open(nil, params.nonce(keyCheckCounter), keyCheck, append(append([]byte{}, header...), keyCheckAdditionalData...)); err != nil {
		return nil, ErrBackupWrongPassphrase
	}

	return &decryptingReader{
		reader: bufferedReader,
		aead:   aead,
		params: params,
		header: header,
	}, nil
}

// Read reads and decrypts data
func (r *decryptingReader) Read(data []byte) (int, error) {
	for len(r.buffer) == 0 {
		if r.finished {
			return 0, io.EOF
		}

		if err := r.readChunk(); err != nil {
			return 0, err
		}
	}

	size := copy(data, r.buffer)
	r.buffer = r.buffer[size:]

	return size, nil
}

func (r *decryptingReader) readChunk() error {
	record := make([]byte, 5)

	if _, err := io.ReadFull(r.reader, record); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return ErrBackupTruncated
		}

		return err
	}

	if record[0] > 1 {
		return fmt.Errorf("%w: invalid encrypted chunk", ErrBackupCorrupted)
	}

	final := record[0] == 1
	length := binary.BigEndian.Uint32(record[1:5])

	if length < encryptionTagSize || length > r.params.chunkSize+encryptionTagSize {
		return fmt.Errorf("%w: invalid encrypted chunk size", ErrBackupCorrupted)
	}

	ciphertext := make([]byte, length)

	if _, err := io.ReadFull(r.reader, ciphertext); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return ErrBackupTruncated
		}

		return err
	}

	plaintext, err := r.aead.Open(nil, r.params.nonce(r.counter), ciphertext, chunkAdditionalData(r.header, r.counter, final))

	if err != nil {
		return fmt.Errorf("%w: encrypted chunk %d authentication failed", ErrBackupCorrupted, r.counter)
	}

	r.counter++
	r.buffer = plaintext

	if final {
		r.finished = true

		// make sure there is no extra data after the final chunk
		extra, err := r.reader.Peek(1)

		if len(extra) > 0 {
			return fmt.Errorf("%w: unexpected data after the final encrypted chunk", ErrBackupCorrupted)
		} else if err != nil && err != io.EOF {
			return err
		}
	}

	return nil
}
