package artifacts

import "errors"

var (
	// ErrArtifactNotFound is returned when an artifact is not found
	ErrArtifactNotFound = errors.New("artifact not found")

	// ErrUploadFailed is returned when upload fails
	ErrUploadFailed = errors.New("upload failed")

	// ErrDownloadFailed is returned when download fails
	ErrDownloadFailed = errors.New("download failed")

	// ErrChecksumMismatch is returned when checksums don't match
	ErrChecksumMismatch = errors.New("checksum mismatch")

	// ErrCompressionFailed is returned when compression fails
	ErrCompressionFailed = errors.New("compression failed")

	// ErrDecompressionFailed is returned when decompression fails
	ErrDecompressionFailed = errors.New("decompression failed")
)
