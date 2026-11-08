package metainforequester

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeUmMessage(buf *bytes.Buffer, header string, payload []byte) {
	msg := append([]byte{20, 1}, header...)
	msg = append(msg, payload...)

	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(msg)))
	buf.Write(length)
	buf.Write(msg)
}

func dataHeader(piece, totalSize int) string {
	return fmt.Sprintf("d8:msg_typei1e5:piecei%de10:total_sizei%dee", piece, totalSize)
}

func TestReadAllPieces(t *testing.T) {
	t.Parallel()

	metadata := bytes.Repeat([]byte("abcdefgh"), (metadataPieceSize*2+100)/8)
	size := len(metadata)

	var buf bytes.Buffer

	// pieces may arrive out of order, interleaved with requests and duplicates
	writeUmMessage(&buf, dataHeader(2, size), metadata[2*metadataPieceSize:])
	writeUmMessage(&buf, "d8:msg_typei0e5:piecei0ee", nil)
	writeUmMessage(&buf, dataHeader(0, size), metadata[:metadataPieceSize])
	writeUmMessage(&buf, dataHeader(0, size), metadata[:metadataPieceSize])
	writeUmMessage(&buf, dataHeader(1, size), metadata[metadataPieceSize:2*metadataPieceSize])

	result, err := readAllPieces(&buf, uint(size))
	require.NoError(t, err)
	assert.Equal(t, metadata, result)
}

func TestReadAllPiecesMalformed(t *testing.T) {
	t.Parallel()

	twoPieces := metadataPieceSize + 100

	for name, tc := range map[string]struct {
		size    int
		header  string
		payload []byte
	}{
		"piece index out of range": {
			size:    100,
			header:  "d8:msg_typei1e5:piecei1ee",
			payload: bytes.Repeat([]byte("x"), 100),
		},
		"negative piece index": {
			size:    100,
			header:  "d8:msg_typei1e5:piecei-1ee",
			payload: bytes.Repeat([]byte("x"), 100),
		},
		"missing piece index": {
			size:    100,
			header:  "d8:msg_typei1ee",
			payload: bytes.Repeat([]byte("x"), 100),
		},
		"short non-final piece": {
			size:    twoPieces,
			header:  dataHeader(0, twoPieces),
			payload: bytes.Repeat([]byte("x"), 100),
		},
		"long final piece": {
			size:    twoPieces,
			header:  dataHeader(1, twoPieces),
			payload: bytes.Repeat([]byte("x"), 200),
		},
		"total size mismatch": {
			size:    100,
			header:  dataHeader(0, 200),
			payload: bytes.Repeat([]byte("x"), 100),
		},
		"reject": {
			size:   100,
			header: "d8:msg_typei2e5:piecei0ee",
		},
		"unknown msg type": {
			size:   100,
			header: "d8:msg_typei9e5:piecei0ee",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			writeUmMessage(&buf, tc.header, tc.payload)

			require.NotPanics(t, func() {
				_, err := readAllPieces(&buf, uint(tc.size))
				assert.Error(t, err)
			})
		})
	}
}

func TestReadAllPiecesConflictingDuplicate(t *testing.T) {
	t.Parallel()

	size := metadataPieceSize + 100

	var buf bytes.Buffer

	writeUmMessage(&buf, dataHeader(0, size), bytes.Repeat([]byte("a"), metadataPieceSize))
	writeUmMessage(&buf, dataHeader(0, size), bytes.Repeat([]byte("b"), metadataPieceSize))

	_, err := readAllPieces(&buf, uint(size))
	assert.ErrorContains(t, err, "conflicts")
}

type failingWriter struct {
	bytes.Buffer
}

func (*failingWriter) Write([]byte) (int, error) {
	return 0, assert.AnError
}

func TestExHandshakeWriteError(t *testing.T) {
	t.Parallel()

	_, _, err := exHandshake(&failingWriter{})
	assert.ErrorIs(t, err, assert.AnError)
}
