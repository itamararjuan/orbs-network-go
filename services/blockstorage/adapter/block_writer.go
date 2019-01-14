package adapter

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"io"
	"sync"
)

type writerSyncer interface {
	io.Writer
	Sync() error
}

type blockWriter struct {
	sync.Mutex
	ws    writerSyncer
	codec blockCodec
}

func newBlockWriter(ws writerSyncer, codec blockCodec) *blockWriter {
	return &blockWriter{
		ws:    ws,
		codec: codec,
	}
}

func (bw *blockWriter) writeBlock(blockPair *protocol.BlockPairContainer) (int, error) {
	bytes, err := bw.codec.encode(blockPair, bw.ws)
	if err != nil {
		return 0, errors.Wrap(err, "failed to write block")
	}

	err = bw.ws.Sync()
	if err != nil {
		return 0, errors.Wrap(err, "failed to flush blocks to disk")
	}

	return bytes, nil
}
