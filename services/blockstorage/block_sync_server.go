// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package blockstorage

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/gossip"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
)

func (s *Service) HandleBlockAvailabilityRequest(ctx context.Context, input *gossiptopics.BlockAvailabilityRequestInput) (*gossiptopics.EmptyOutput, error) {
	err := s.sourceHandleBlockAvailabilityRequest(ctx, input.Message)
	return nil, err
}

func (s *Service) HandleBlockSyncRequest(ctx context.Context, input *gossiptopics.BlockSyncRequestInput) (*gossiptopics.EmptyOutput, error) {
	err := s.sourceHandleBlockSyncRequest(ctx, input.Message)
	return nil, err
}

func (s *Service) sourceHandleBlockAvailabilityRequest(ctx context.Context, message *gossipmessages.BlockAvailabilityRequestMessage) error {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	logger.Info("received block availability request",
		log.Stringable("petitioner", message.Sender.SenderNodeAddress()),
		log.Uint64("requested-first-block", uint64(message.SignedBatchRange.FirstBlockHeight())),
		log.Uint64("requested-last-block", uint64(message.SignedBatchRange.LastBlockHeight())),
		log.Uint64("requested-last-committed-block", uint64(message.SignedBatchRange.LastCommittedBlockHeight())))

	out, err := s.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
	if err != nil {
		return err
	}
	lastCommittedBlockHeight := out.LastCommittedBlockHeight

	if lastCommittedBlockHeight <= message.SignedBatchRange.LastCommittedBlockHeight() {
		return nil
	}

	firstAvailableBlockHeight := primitives.BlockHeight(1)
	blockType := message.SignedBatchRange.BlockType()

	response := &gossiptopics.BlockAvailabilityResponseInput{
		RecipientNodeAddress: message.Sender.SenderNodeAddress(),
		Message: &gossipmessages.BlockAvailabilityResponseMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderNodeAddress: s.config.NodeAddress(),
			}).Build(),
			SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
				BlockType:                blockType,
				LastBlockHeight:          lastCommittedBlockHeight,
				FirstBlockHeight:         firstAvailableBlockHeight,
				LastCommittedBlockHeight: lastCommittedBlockHeight,
			}).Build(),
		},
	}

	logger.Info("sending the response for availability request",
		log.Stringable("petitioner", response.RecipientNodeAddress),
		log.Uint64("first-available-block-height", uint64(response.Message.SignedBatchRange.FirstBlockHeight())),
		log.Uint64("last-available-block-height", uint64(response.Message.SignedBatchRange.LastBlockHeight())),
		log.Uint64("last-committed-available-block-height", uint64(response.Message.SignedBatchRange.LastCommittedBlockHeight())),
		log.Stringable("source", response.Message.Sender.SenderNodeAddress()),
	)

	_, err = s.gossip.SendBlockAvailabilityResponse(ctx, response)
	return err
}

func (s *Service) sourceHandleBlockSyncRequest(ctx context.Context, message *gossipmessages.BlockSyncRequestMessage) error {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	senderNodeAddress := message.Sender.SenderNodeAddress()
	blockType := message.SignedChunkRange.BlockType()
	firstRequestedBlockHeight := message.SignedChunkRange.FirstBlockHeight()
	lastRequestedBlockHeight := message.SignedChunkRange.LastBlockHeight()

	out, err := s.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
	if err != nil {
		return err
	}
	lastCommittedBlockHeight := out.LastCommittedBlockHeight

	logger.Info("received block sync request",
		log.Stringable("petitioner", message.Sender.SenderNodeAddress()),
		log.Uint64("first-requested-block-height", uint64(firstRequestedBlockHeight)),
		log.Uint64("last-requested-block-height", uint64(lastRequestedBlockHeight)),
		log.Uint64("last-committed-block-height", uint64(lastCommittedBlockHeight)))

	if firstRequestedBlockHeight > lastCommittedBlockHeight {
		return errors.New("firstBlockHeight is greater than lastCommittedBlockHeight")
	}

	if firstRequestedBlockHeight-lastCommittedBlockHeight > primitives.BlockHeight(s.config.BlockSyncNumBlocksInBatch()-1) {
		lastRequestedBlockHeight = firstRequestedBlockHeight + primitives.BlockHeight(s.config.BlockSyncNumBlocksInBatch()-1)
	}

	blocks, firstAvailableBlockHeight, _, err := s.GetBlockSlice(firstRequestedBlockHeight, lastRequestedBlockHeight)
	if err != nil {
		return errors.Wrap(err, "block sync failed reading from block persistence")
	}

	chunkSize := uint(len(blocks))
	for {
		lastAvailableBlockHeight := firstAvailableBlockHeight + primitives.BlockHeight(chunkSize) - 1
		logger.Info("sending blocks to another node via block sync",
			log.Stringable("petitioner", senderNodeAddress),
			log.Uint64("first-available-block-height", uint64(firstAvailableBlockHeight)),
			log.Uint64("last-available-block-height", uint64(lastAvailableBlockHeight)))

		response := &gossiptopics.BlockSyncResponseInput{
			RecipientNodeAddress: senderNodeAddress,
			Message: &gossipmessages.BlockSyncResponseMessage{
				Sender: (&gossipmessages.SenderSignatureBuilder{
					SenderNodeAddress: s.config.NodeAddress(),
				}).Build(),
				SignedChunkRange: (&gossipmessages.BlockSyncRangeBuilder{
					BlockType:                blockType,
					FirstBlockHeight:         firstAvailableBlockHeight,
					LastBlockHeight:          lastAvailableBlockHeight,
					LastCommittedBlockHeight: lastCommittedBlockHeight,
				}).Build(),
				BlockPairs: blocks[:chunkSize],
			},
		}
		_, err = s.gossip.SendBlockSyncResponse(ctx, response)
		if err != nil {
			if !gossip.IsChunkTooBigError(err) { // A non chunk-size related error, return immediately
				return err
			}
			if chunkSize == 0 { // We just tried sending a zero-length chunk and failed, time to give up
				return err
			}
			chunkSize /= 2 // try again with a smaller chunk
			continue
		}

		return nil
	}
}
