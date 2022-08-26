package encode

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Encoding is the encoding type for client.Object.
// For now, yaml is the only supported encoding type.
type Encoding string

// Encoder is an interface for encoding a kubernetes object to certain type.
type Encoder interface {
	// Encode takes a kubernetes object and returns an array of bytes.
	Encode(client.Object) ([]byte, error)
	// Extension returns the extention name for the encoder.
	Extension() string
}
