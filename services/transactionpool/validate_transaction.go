package transactionpool

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"time"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
)

const ProtocolVersion = primitives.ProtocolVersion(1)

type validator func(transaction *protocol.SignedTransaction) error

type validationContext struct {
	expiryWindow time.Duration
	lastCommittedBlockTimestamp primitives.TimestampNano
	futureTimestampGrace time.Duration
}

func validateTransaction(transaction *protocol.SignedTransaction, vctx validationContext) error {
	//TODO can we create the list of validators once on system startup; this will save on performance in the critical path
	validators := []validator {
		validateProtocolVersion,
		validateSignerAndContractName,
		validateTransactionNotExpired(vctx),
		validateTransactionNotInFuture(vctx),
	}

	for _, validate := range validators {
		err := validate(transaction)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateProtocolVersion(tx *protocol.SignedTransaction) error {
	if tx.Transaction().ProtocolVersion() != ProtocolVersion {
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_UNSUPPORTED_VERSION}
	}
	return nil
}

func validateSignerAndContractName(transaction *protocol.SignedTransaction) error {
	tx := transaction.Transaction()
	if tx.ContractName() == "" ||
		!tx.Signer().IsSchemeEddsa() ||
		len(tx.Signer().Eddsa().SignerPublicKey()) != signature.ED25519_PUBLIC_KEY_SIZE {
		//TODO is this the correct status?
		return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH}
	}
	return nil
}

func validateTransactionNotExpired(vctx validationContext) validator {
	return func(transaction *protocol.SignedTransaction) error {
		if time.Unix(0, int64(transaction.Transaction().Timestamp())).Before(time.Now().Add(vctx.expiryWindow * -1)) {
			return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_TIME_STAMP_WINDOW_EXCEEDED}
		}

		return nil
	}
}

func validateTransactionNotInFuture(vctx validationContext) validator {
	return func(transaction *protocol.SignedTransaction) error {
		if transaction.Transaction().Timestamp() > vctx.lastCommittedBlockTimestamp + primitives.TimestampNano(vctx.futureTimestampGrace.Nanoseconds()) {
			return &ErrTransactionRejected{protocol.TRANSACTION_STATUS_REJECTED_TIME_STAMP_WINDOW_EXCEEDED}
		}

		return nil
	}
}