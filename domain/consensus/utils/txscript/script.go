// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/kaspanet/kaspad/domain/consensus/utils/constants"

	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/consensushashing"
	"github.com/pkg/errors"
)

// SigHashType represents hash type bits at the end of a signature.
type SigHashType uint32

// Hash type bits from the end of a signature.
const (
	SigHashOld          SigHashType = 0x0
	SigHashAll          SigHashType = 0x1
	SigHashNone         SigHashType = 0x2
	SigHashSingle       SigHashType = 0x3
	SigHashAnyOneCanPay SigHashType = 0x80

	// sigHashMask defines the number of bits of the hash type which is used
	// to identify which outputs are signed.
	sigHashMask = 0x1f
)

// These are the constants specified for maximums in individual scripts.
const (
	MaxOpsPerScript       = 201 // Max number of non-push operations.
	MaxPubKeysPerMultiSig = 20  // Multisig can't have more sigs than this.
	MaxScriptElementSize  = 520 // Max bytes pushable to the stack.
)

// isSmallInt returns whether or not the opcode is considered a small integer,
// which is an OP_0, or OP_1 through OP_16.
func isSmallInt(op *opcode) bool {
	if op.value == Op0 || (op.value >= Op1 && op.value <= Op16) {
		return true
	}
	return false
}

// isScriptHash returns true if the script passed is a pay-to-script-hash
// transaction, false otherwise.
func isScriptHash(pops []parsedOpcode) bool {
	return len(pops) == 3 &&
		pops[0].opcode.value == OpHash160 &&
		pops[1].opcode.value == OpData20 &&
		pops[2].opcode.value == OpEqual
}

// IsPayToScriptHash returns true if the script is in the standard
// pay-to-script-hash (P2SH) format, false otherwise.
func IsPayToScriptHash(script *externalapi.ScriptPublicKey) bool {
	pops, err := parseScript(script.Script)
	if err != nil {
		return false
	}
	return isScriptHash(pops)
}

// isPushOnly returns true if the script only pushes data, false otherwise.
func isPushOnly(pops []parsedOpcode) bool {
	// NOTE: This function does NOT verify opcodes directly since it is
	// internal and is only called with parsed opcodes for scripts that did
	// not have any parse errors. Thus, consensus is properly maintained.

	for _, pop := range pops {
		// All opcodes up to OP_16 are data push instructions.
		// NOTE: This does consider OP_RESERVED to be a data push
		// instruction, but execution of OP_RESERVED will fail anyways
		// and matches the behavior required by consensus.
		if pop.opcode.value > Op16 {
			return false
		}
	}
	return true
}

// parseScriptTemplate is the same as parseScript but allows the passing of the
// template list for testing purposes. When there are parse errors, it returns
// the list of parsed opcodes up to the point of failure along with the error.
func parseScriptTemplate(script []byte, opcodes *[256]opcode) ([]parsedOpcode, error) {
	retScript := make([]parsedOpcode, 0, len(script))
	for i := 0; i < len(script); {
		instr := script[i]
		op := &opcodes[instr]
		pop := parsedOpcode{opcode: op}

		// Parse data out of instruction.
		switch {
		// No additional data. Note that some of the opcodes, notably
		// OP_1NEGATE, OP_0, and OP_[1-16] represent the data
		// themselves.
		case op.length == 1:
			i++

		// Data pushes of specific lengths -- OP_DATA_[1-75].
		case op.length > 1:
			if len(script[i:]) < op.length {
				str := fmt.Sprintf("opcode %s requires %d "+
					"bytes, but script only has %d remaining",
					op.name, op.length, len(script[i:]))
				return retScript, scriptError(ErrMalformedPush,
					str)
			}

			// Slice out the data.
			pop.data = script[i+1 : i+op.length]
			i += op.length

		// Data pushes with parsed lengths -- OP_PUSHDATAP{1,2,4}.
		case op.length < 0:
			var l uint
			off := i + 1

			if len(script[off:]) < -op.length {
				str := fmt.Sprintf("opcode %s requires %d "+
					"bytes, but script only has %d remaining",
					op.name, -op.length, len(script[off:]))
				return retScript, scriptError(ErrMalformedPush,
					str)
			}

			// Next -length bytes are little endian length of data.
			switch op.length {
			case -1:
				l = uint(script[off])
			case -2:
				l = ((uint(script[off+1]) << 8) |
					uint(script[off]))
			case -4:
				l = ((uint(script[off+3]) << 24) |
					(uint(script[off+2]) << 16) |
					(uint(script[off+1]) << 8) |
					uint(script[off]))
			default:
				str := fmt.Sprintf("invalid opcode length %d",
					op.length)
				return retScript, scriptError(ErrMalformedPush,
					str)
			}

			// Move offset to beginning of the data.
			off += -op.length

			// Disallow entries that do not fit script or were
			// sign extended.
			if int(l) > len(script[off:]) || int(l) < 0 {
				str := fmt.Sprintf("opcode %s pushes %d bytes, "+
					"but script only has %d remaining",
					op.name, int(l), len(script[off:]))
				return retScript, scriptError(ErrMalformedPush,
					str)
			}

			pop.data = script[off : off+int(l)]
			i += 1 - op.length + int(l)
		}

		retScript = append(retScript, pop)
	}

	return retScript, nil
}

// parseScript preparses the script in bytes into a list of parsedOpcodes while
// applying a number of sanity checks.
func parseScript(script []byte) ([]parsedOpcode, error) {
	return parseScriptTemplate(script, &opcodeArray)
}

// unparseScript reversed the action of parseScript and returns the
// parsedOpcodes as a list of bytes
func unparseScript(pops []parsedOpcode) ([]byte, error) {
	script := make([]byte, 0, len(pops))
	for _, pop := range pops {
		b, err := pop.bytes()
		if err != nil {
			return nil, err
		}
		script = append(script, b...)
	}
	return script, nil
}

// DisasmString formats a disassembled script for one line printing. When the
// script fails to parse, the returned string will contain the disassembled
// script up to the point the failure occurred along with the string '[error]'
// appended. In addition, the reason the script failed to parse is returned
// if the caller wants more information about the failure.
func DisasmString(version uint16, buf []byte) (string, error) {
	// currently, there is only one version exists so it equals to the max version.
	if version == constants.MaxScriptPublicKeyVersion {
		var disbuf bytes.Buffer
		opcodes, err := parseScript(buf)
		for _, pop := range opcodes {
			disbuf.WriteString(pop.print(true))
			disbuf.WriteByte(' ')
		}
		if disbuf.Len() > 0 {
			disbuf.Truncate(disbuf.Len() - 1)
		}
		if err != nil {
			disbuf.WriteString("[error]")
		}
		return disbuf.String(), err
	}
	return "", scriptError(ErrPubKeyFormat, "the version of the scriptPublicHash is higher then the known version")
}

// canonicalPush returns true if the object is either not a push instruction
// or the push instruction contained wherein is matches the canonical form
// or using the smallest instruction to do the job. False otherwise.
func canonicalPush(pop parsedOpcode) bool {
	opcode := pop.opcode.value
	data := pop.data
	dataLen := len(pop.data)
	if opcode > Op16 {
		return true
	}

	if opcode < OpPushData1 && opcode > Op0 && (dataLen == 1 && data[0] <= 16) {
		return false
	}
	if opcode == OpPushData1 && dataLen < OpPushData1 {
		return false
	}
	if opcode == OpPushData2 && dataLen <= 0xff {
		return false
	}
	if opcode == OpPushData4 && dataLen <= 0xffff {
		return false
	}
	return true
}

// shallowCopyTx creates a shallow copy of the transaction for use when
// calculating the signature hash. It is used over the Copy method on the
// transaction itself since that is a deep copy and therefore does more work and
// allocates much more space than needed.
func shallowCopyTx(tx *externalapi.DomainTransaction) externalapi.DomainTransaction {
	// As an additional memory optimization, use contiguous backing arrays
	// for the copied inputs and outputs and point the final slice of
	// pointers into the contiguous arrays. This avoids a lot of small
	// allocations.
	txCopy := externalapi.DomainTransaction{
		Version:      tx.Version,
		Inputs:       make([]*externalapi.DomainTransactionInput, len(tx.Inputs)),
		Outputs:      make([]*externalapi.DomainTransactionOutput, len(tx.Outputs)),
		LockTime:     tx.LockTime,
		SubnetworkID: tx.SubnetworkID,
		Gas:          tx.Gas,
		PayloadHash:  tx.PayloadHash,
		Payload:      tx.Payload,
	}
	txIns := make([]externalapi.DomainTransactionInput, len(tx.Inputs))
	for i, oldTxIn := range tx.Inputs {
		txIns[i] = *oldTxIn
		txCopy.Inputs[i] = &txIns[i]
	}
	txOuts := make([]externalapi.DomainTransactionOutput, len(tx.Outputs))
	for i, oldTxOut := range tx.Outputs {
		txOuts[i] = *oldTxOut
		txCopy.Outputs[i] = &txOuts[i]
	}
	return txCopy
}

// CalcSignatureHash will, given a script and hash type for the current script
// engine instance, calculate the signature hash to be used for signing and
// verification.
func CalcSignatureHash(script *externalapi.ScriptPublicKey, hashType SigHashType, tx *externalapi.DomainTransaction, idx int) (*externalapi.DomainHash, error) {
	if script.Version > constants.MaxScriptPublicKeyVersion {
		return nil, errors.Errorf("Script version is unkown.")
	}
	parsedScript, err := parseScript(script.Script)
	if err != nil {
		return nil, errors.Errorf("cannot parse output script: %s", err)
	}
	return calcSignatureHash(parsedScript, script.Version, hashType, tx, idx)
}

// calcSignatureHash will, given a script and hash type for the current script
// engine instance, calculate the signature hash to be used for signing and
// verification.
func calcSignatureHash(prevScriptPublicKey []parsedOpcode, scriptVersion uint16, hashType SigHashType, tx *externalapi.DomainTransaction, idx int) (*externalapi.DomainHash, error) {
	// The SigHashSingle signature type signs only the corresponding input
	// and output (the output with the same index number as the input).
	//
	// Since transactions can have more inputs than outputs, this means it
	// is improper to use SigHashSingle on input indices that don't have a
	// corresponding output.
	if hashType&sigHashMask == SigHashSingle && idx >= len(tx.Outputs) {
		return nil, scriptError(ErrInvalidSigHashSingleIndex, "sigHashSingle index out of bounds")
	}

	// Make a shallow copy of the transaction, zeroing out the payload and the
	// script for all inputs that are not currently being processed.
	txCopy := shallowCopyTx(tx)
	txCopy.Payload = []byte{}
	for i := range txCopy.Inputs {
		if i == idx {
			// UnparseScript cannot fail here because removeOpcode
			// above only returns a valid script.
			sigScript, _ := unparseScript(prevScriptPublicKey)
			var version [2]byte
			binary.LittleEndian.PutUint16(version[:], scriptVersion)
			txCopy.Inputs[idx].SignatureScript = append(version[:], sigScript...)
		} else {
			txCopy.Inputs[i].SignatureScript = nil
		}
	}

	switch hashType & sigHashMask {
	case SigHashNone:
		txCopy.Outputs = txCopy.Outputs[0:0] // Empty slice.
		for i := range txCopy.Inputs {
			if i != idx {
				txCopy.Inputs[i].Sequence = 0
			}
		}

	case SigHashSingle:
		// Resize output array to up to and including requested index.
		txCopy.Outputs = txCopy.Outputs[:idx+1]

		// All but current output get zeroed out.
		for i := 0; i < idx; i++ {
			txCopy.Outputs[i].Value = 0
			txCopy.Outputs[i].ScriptPublicKey.Script = nil
			txCopy.Outputs[i].ScriptPublicKey.Version = 0
		}

		// Sequence on all other inputs is 0, too.
		for i := range txCopy.Inputs {
			if i != idx {
				txCopy.Inputs[i].Sequence = 0
			}
		}

	default:
		// Consensus treats undefined hashtypes like normal SigHashAll
		// for purposes of hash generation.
		fallthrough
	case SigHashOld:
		fallthrough
	case SigHashAll:
		// Nothing special here.
	}
	if hashType&SigHashAnyOneCanPay != 0 {
		txCopy.Inputs = txCopy.Inputs[idx : idx+1]
	}

	// The final hash is the hash of both the serialized modified
	// transaction and the hash type (encoded as a 4-byte little-endian
	// value) appended.
	return consensushashing.TransactionHashForSigning(&txCopy, uint32(hashType)), nil
}

// asSmallInt returns the passed opcode, which must be true according to
// isSmallInt(), as an integer.
func asSmallInt(op *opcode) int {
	if op.value == Op0 {
		return 0
	}

	return int(op.value - (Op1 - 1))
}

// getSigOpCount is the implementation function for counting the number of
// signature operations in the script provided by pops. If precise mode is
// requested then we attempt to count the number of operations for a multisig
// op. Otherwise we use the maximum.
func getSigOpCount(pops []parsedOpcode, precise bool) int {
	nSigs := 0
	for i, pop := range pops {
		switch pop.opcode.value {
		case OpCheckSig:
			fallthrough
		case OpCheckSigVerify:
			nSigs++
		case OpCheckMultiSig:
			fallthrough
		case OpCheckMultiSigVerify:
			// If we are being precise then look for familiar
			// patterns for multisig, for now all we recognize is
			// OP_1 - OP_16 to signify the number of pubkeys.
			// Otherwise, we use the max of 20.
			if precise && i > 0 &&
				pops[i-1].opcode.value >= Op1 &&
				pops[i-1].opcode.value <= Op16 {
				nSigs += asSmallInt(pops[i-1].opcode)
			} else {
				nSigs += MaxPubKeysPerMultiSig
			}
		default:
			// Not a sigop.
		}
	}

	return nSigs
}

// GetSigOpCount provides a quick count of the number of signature operations
// in a script. a CHECKSIG operations counts for 1, and a CHECK_MULTISIG for 20.
// If the script fails to parse, then the count up to the point of failure is
// returned.
func GetSigOpCount(script []byte) int {
	// Don't check error since parseScript returns the parsed-up-to-error
	// list of pops.
	pops, _ := parseScript(script)
	return getSigOpCount(pops, false)
}

// GetPreciseSigOpCount returns the number of signature operations in
// scriptPubKey. If p2sh is true then scriptSig may be searched for the
// Pay-To-Script-Hash script in order to find the precise number of signature
// operations in the transaction. If the script fails to parse, then the count
// up to the point of failure is returned.
func GetPreciseSigOpCount(scriptSig []byte, scriptPubKey *externalapi.ScriptPublicKey, isP2SH bool) int {
	// Don't check error since parseScript returns the parsed-up-to-error
	// list of pops.
	pops, _ := parseScript(scriptPubKey.Script)

	// Treat non P2SH transactions as normal.
	if !(isP2SH && isScriptHash(pops)) {
		return getSigOpCount(pops, true)
	}

	// The public key script is a pay-to-script-hash, so parse the signature
	// script to get the final item. Scripts that fail to fully parse count
	// as 0 signature operations.
	sigPops, err := parseScript(scriptSig)
	if err != nil {
		return 0
	}

	// The signature script must only push data to the stack for P2SH to be
	// a valid pair, so the signature operation count is 0 when that is not
	// the case.
	if !isPushOnly(sigPops) || len(sigPops) == 0 {
		return 0
	}

	// The P2SH script is the last item the signature script pushes to the
	// stack. When the script is empty, there are no signature operations.
	shScript := sigPops[len(sigPops)-1].data
	if len(shScript) == 0 {
		return 0
	}

	// Parse the P2SH script and don't check the error since parseScript
	// returns the parsed-up-to-error list of pops and the consensus rules
	// dictate signature operations are counted up to the first parse
	// failure.
	shPops, _ := parseScript(shScript)
	return getSigOpCount(shPops, true)
}

// IsUnspendable returns whether the passed public key script is unspendable, or
// guaranteed to fail at execution. This allows inputs to be pruned instantly
// when entering the UTXO set.
func IsUnspendable(scriptPubKey []byte) bool {
	pops, err := parseScript(scriptPubKey)
	if err != nil {
		return true
	}

	return len(pops) > 0 && pops[0].opcode.value == OpReturn
}
