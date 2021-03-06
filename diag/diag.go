package diag

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
)

const (
	// MaxUploadBatchSize is the maximum amount of diagnosis keys to be
	// uploaded per request.
	MaxUploadBatchSize = 14

	// DiagnosisKeySize is the size of a `Diagnosis Key`, consisting of a
	// `TemporaryExposureKey` (16 bytes) and a `ENIntervalNumber` (4 bytes).
	DiagnosisKeySize = 20

	// UploadLimit is the size limit for uploading diagnosis keys in bytes.
	UploadLimit = MaxUploadBatchSize * DiagnosisKeySize
)

var (
	// ErrNilDiagKeys is used when an empty diagnosis keyset is used.
	ErrNilDiagKeys = errors.New("diag: diagnosis key array cannot be empty")

	// ErrMaxUploadExceeded is used when upload batch size exceeds the limit.
	ErrMaxUploadExceeded = errors.New("diag: maximum upload batch size exceeded")
)

// DiagnosisKey is the combination of a `TemporaryExposureKey` and its related
// `ENIntervalNumber`. In total, a DiagnosisKey takes up 20 bytes when sent over
// the wire. Note: The `ENIntervalNumber` is the 10 minute time window since Unix
// Epoch when the key `TemporaryExposureKey` was generated.
// @see https://covid19-static.cdn-apple.com/applications/covid19/current/static/contact-tracing/pdf/ExposureNotification-CryptographySpecificationv1.1.pdf
type DiagnosisKey struct {
	TemporaryExposureKey [16]byte
	ENIntervalNumber     uint32
}

// Repository defines an interface for storing and retrieving diagnosis keys
// in a repository.
type Repository interface {
	StoreDiagnosisKeys(context.Context, []DiagnosisKey) error
	FindAllDiagnosisKeys(context.Context) ([]DiagnosisKey, error)
}

// Service represents the service for managing diagnosis keys.
type Service struct {
	repo Repository
}

// NewService returns a new Service.
func NewService(repo Repository) Service {
	return Service{repo: repo}
}

// StoreDiagnosisKeys persists a set of diagnosis keys to the repository.
func (s Service) StoreDiagnosisKeys(ctx context.Context, diagKeys []DiagnosisKey) error {
	return s.repo.StoreDiagnosisKeys(ctx, diagKeys)
}

// FindAllDiagnosisKeys fetches all diagnosis keys from the repository.
func (s Service) FindAllDiagnosisKeys(ctx context.Context) ([]DiagnosisKey, error) {
	return s.repo.FindAllDiagnosisKeys(ctx)
}

// ParseDiagnosisKeys reads and parses diagnosis keys from an io.Reader.
func (s Service) ParseDiagnosisKeys(r io.Reader) ([]DiagnosisKey, error) {
	buf := make([]byte, UploadLimit+1)
	n, err := r.Read(buf)

	switch {
	case err != nil && err != io.EOF:
		return nil, err
	case n == 0:
		return nil, io.ErrUnexpectedEOF
	case n == UploadLimit+1:
		return nil, ErrMaxUploadExceeded
	case n%DiagnosisKeySize != 0:
		return nil, io.ErrUnexpectedEOF
	}

	keyCount := n / DiagnosisKeySize
	diagKeys := make([]DiagnosisKey, keyCount)

	for i := 0; i < keyCount; i++ {
		start := i * DiagnosisKeySize
		var key [16]byte
		copy(key[:], buf[start:start+16])
		enin := binary.BigEndian.Uint32(buf[start+16 : start+DiagnosisKeySize])

		diagKeys[i] = DiagnosisKey{TemporaryExposureKey: key, ENIntervalNumber: enin}
	}

	return diagKeys, nil
}
