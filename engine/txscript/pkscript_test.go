package txscript

import (
	"bytes"
	"testing"
)

// TestComputePkScript ensures that we can correctly re-derive an output's
// pkScript by looking at the input's signature script/witness attempting to
// spend it.
func TestComputePkScript(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sigScript []byte
		witness   TxWitness
		class     ScriptClass
		pkScript  []byte
	}{
		{
			name:      "empty sigScript and witness",
			sigScript: nil,
			witness:   nil,
			class:     NonStandardTy,
			pkScript:  nil,
		},
		{
			name: "P2PKH sigScript",
			sigScript: []byte{
				// OP_DATA_73,
				0x49,
				// <73-byte sig>
				0x30, 0x44, 0x02, 0x20, 0x65, 0x92, 0xd8, 0x8e,
				0x1d, 0x0a, 0x4a, 0x3c, 0xc5, 0x9f, 0x92, 0xae,
				0xfe, 0x62, 0x54, 0x74, 0xa9, 0x4d, 0x13, 0xa5,
				0x9f, 0x84, 0x97, 0x78, 0xfc, 0xe7, 0xdf, 0x4b,
				0xe0, 0xc2, 0x28, 0xd8, 0x02, 0x20, 0x2d, 0xea,
				0x36, 0x96, 0x19, 0x1f, 0xb7, 0x00, 0xc5, 0xa7,
				0x7e, 0x22, 0xd9, 0xfb, 0x6b, 0x42, 0x67, 0x42,
				0xa4, 0x2c, 0xac, 0xdb, 0x74, 0xa2, 0x7c, 0x43,
				0xcd, 0x89, 0xa0, 0xf9, 0x44, 0x54, 0x12, 0x74,
				0x01,
				// OP_DATA_33
				0x21,
				// <33-byte compressed pubkey>
				0x02, 0x7d, 0x56, 0x12, 0x09, 0x75, 0x31, 0xc2,
				0x17, 0xfd, 0xd4, 0xd2, 0xe1, 0x7a, 0x35, 0x4b,
				0x17, 0xf2, 0x7a, 0xef, 0x30, 0x9f, 0xb2, 0x7f,
				0x1f, 0x1f, 0x7b, 0x73, 0x7d, 0x9a, 0x24, 0x49,
				0x90,
			},
			witness: nil,
			class:   PubKeyHashTy,
			pkScript: []byte{
				// OP_DUP
				0x76,
				// OP_HASH160
				0xa9,
				// OP_DATA_20
				0x14,
				// <20-byte pubkey hash>
				0x67, 0x8a, 0x56, 0x45, 0xcc, 0x5c, 0x5d, 0x1b,
				0x13, 0x67, 0x69, 0xd0, 0x37, 0xad, 0xd2, 0x2f,
				0x9e, 0x7d, 0x9, 0x82,
				// OP_EQUALVERIFY
				0x88,
				// OP_CHECKSIG
				0xac,
			},
		},
		{
			name: "NP2WPKH sigScript",
			// Since this is a NP2PKH output, the sigScript is a
			// data push of a serialized v0 P2WPKH script.
			sigScript: []byte{
				// OP_DATA_16
				0x16,
				// <22-byte redeem script>
				0x00, 0x14, 0x1d, 0x7c, 0xd6, 0xc7, 0x5c, 0x2e,
				0x86, 0xf4, 0xcb, 0xf9, 0x8e, 0xae, 0xd2, 0x21,
				0xb3, 0x0b, 0xd9, 0xa0, 0xb9, 0x28,
			},
			// NP2PKH outputs include a witness, but it is not
			// needed to reconstruct the pkScript.
			witness: nil,
			class:   ScriptHashTy,
			pkScript: []byte{
				// OP_HASH160
				0xa9,
				// OP_DATA_20
				0x14,
				// <20-byte script hash>
				0x59, 0xf4, 0x7d, 0xa6, 0x55, 0x1a, 0x80, 0x6f,
				0x65, 0xb3, 0x4d, 0xca, 0x61, 0x13, 0x87, 0xd,
				0x80, 0x4f, 0x1d, 0xbc,
				// OP_EQUAL
				0x87,
			},
		},
		{
			name: "P2SH sigScript",
			sigScript: []byte{
				0x00, 0x49, 0x30, 0x46, 0x02, 0x21, 0x00, 0xda,
				0xe6, 0xb6, 0x14, 0x1b, 0xa7, 0x24, 0x4f, 0x54,
				0x62, 0xb6, 0x2a, 0x3b, 0x27, 0x59, 0xde, 0xe4,
				0x46, 0x76, 0x19, 0x4e, 0x6c, 0x56, 0x8d, 0x5b,
				0x1c, 0xda, 0x96, 0x2d, 0x4f, 0x6d, 0x79, 0x02,
				0x21, 0x00, 0xa6, 0x6f, 0x60, 0x34, 0x46, 0x09,
				0x0a, 0x22, 0x3c, 0xec, 0x30, 0x33, 0xd9, 0x86,
				0x24, 0xd2, 0x73, 0xa8, 0x91, 0x55, 0xa5, 0xe6,
				0x96, 0x66, 0x0b, 0x6a, 0x50, 0xa3, 0x46, 0x45,
				0xbb, 0x67, 0x01, 0x48, 0x30, 0x45, 0x02, 0x21,
				0x00, 0xe2, 0x73, 0x49, 0xdb, 0x93, 0x82, 0xe1,
				0xf8, 0x8d, 0xae, 0x97, 0x5c, 0x71, 0x19, 0xb7,
				0x79, 0xb6, 0xda, 0x43, 0xa8, 0x4f, 0x16, 0x05,
				0x87, 0x11, 0x9f, 0xe8, 0x12, 0x1d, 0x85, 0xae,
				0xee, 0x02, 0x20, 0x6f, 0x23, 0x2d, 0x0a, 0x7b,
				0x4b, 0xfa, 0xcd, 0x56, 0xa0, 0x72, 0xcc, 0x2a,
				0x44, 0x81, 0x31, 0xd1, 0x0d, 0x73, 0x35, 0xf9,
				0xa7, 0x54, 0x8b, 0xee, 0x1f, 0x70, 0xc5, 0x71,
				0x0b, 0x37, 0x9e, 0x01, 0x47, 0x52, 0x21, 0x03,
				0xab, 0x11, 0x5d, 0xa6, 0xdf, 0x4f, 0x54, 0x0b,
				0xd6, 0xc9, 0xc4, 0xbe, 0x5f, 0xdd, 0xcc, 0x24,
				0x58, 0x8e, 0x7c, 0x2c, 0xaf, 0x13, 0x82, 0x28,
				0xdd, 0x0f, 0xce, 0x29, 0xfd, 0x65, 0xb8, 0x7c,
				0x21, 0x02, 0x15, 0xe8, 0xb7, 0xbf, 0xfe, 0x8d,
				0x9b, 0xbd, 0x45, 0x81, 0xf9, 0xc3, 0xb6, 0xf1,
				0x6d, 0x67, 0x08, 0x36, 0xc3, 0x0b, 0xb2, 0xe0,
				0x3e, 0xfd, 0x9d, 0x41, 0x03, 0xb5, 0x59, 0xeb,
				0x67, 0xcd, 0x52, 0xae,
			},
			witness: nil,
			class:   ScriptHashTy,
			pkScript: []byte{
				// OP_HASH160
				0xA9,
				// OP_DATA_20
				0x14,
				// <20-byte script hash>
				0x37, 0x27, 0xef, 0x45, 0xa, 0xbe, 0xa2, 0x38,
				0x90, 0xc6, 0x5b, 0xad, 0x45, 0x92, 0x5a, 0xf4,
				0xdf, 0xe9, 0xc5, 0xfa,
				// OP_EQUAL
				0x87,
			},
		},
		// Invalid P2SH (non push-data only script).
		{
			name:      "invalid P2SH sigScript",
			sigScript: []byte{0x6b, 0x65, 0x6b}, // kek
			witness:   nil,
			class:     NonStandardTy,
			pkScript:  nil,
		},
		{
			name:      "P2WSH witness",
			sigScript: nil,
			witness: [][]byte{
				{},
				// Witness script.
				{
					0x21, 0x03, 0x82, 0x62, 0xa6, 0xc6,
					0xce, 0xc9, 0x3c, 0x2d, 0x3e, 0xcd,
					0x6c, 0x60, 0x72, 0xef, 0xea, 0x86,
					0xd0, 0x2f, 0xf8, 0xe3, 0x32, 0x8b,
					0xbd, 0x02, 0x42, 0xb2, 0x0a, 0xf3,
					0x42, 0x59, 0x90, 0xac, 0xac,
				},
			},
			class: WitnessV0ScriptHashTy,
			pkScript: []byte{
				// OP_0
				0x00,
				// OP_DATA_32
				0x20,
				// <32-byte script hash>
				0x95, 0xe2, 0xba, 0x1b, 0xdd, 0xb2, 0x55, 0x86,
				0xff, 0x77, 0x3a, 0xb7, 0x16, 0x18, 0xa, 0x1e,
				0x26, 0x6d, 0x67, 0x98, 0x25, 0x34, 0xb5, 0x24,
				0x36, 0x5b, 0xdc, 0x5c, 0x39, 0x8, 0x79, 0x81,
			},
		},
		{
			name:      "P2WPKH witness",
			sigScript: nil,
			witness: [][]byte{
				// Signature is not needed to re-derive the
				// pkScript.
				{},
				// Compressed pubkey.
				{
					0x03, 0x82, 0x62, 0xa6, 0xc6, 0xce,
					0xc9, 0x3c, 0x2d, 0x3e, 0xcd, 0x6c,
					0x60, 0x72, 0xef, 0xea, 0x86, 0xd0,
					0x2f, 0xf8, 0xe3, 0x32, 0x8b, 0xbd,
					0x02, 0x42, 0xb2, 0x0a, 0xf3, 0x42,
					0x59, 0x90, 0xac,
				},
			},
			class: WitnessV0PubKeyHashTy,
			pkScript: []byte{
				// OP_0
				0x00,
				// OP_DATA_20
				0x14,
				// <20-byte pubkey hash>
				0x46, 0x92, 0xa7, 0xa5, 0x11, 0xae, 0xb1, 0x71,
				0xd8, 0x40, 0x0, 0xda, 0x59, 0x39, 0x6e, 0xf9,
				0x55, 0x81, 0x26, 0x6c,
			},
		},
		// Invalid v0 P2WPKH - same as above but missing a byte on the
		// public key.
		{
			name:      "invalid P2WPKH witness",
			sigScript: nil,
			witness: [][]byte{
				// Signature is not needed to re-derive the
				// pkScript.
				{},
				// Malformed compressed pubkey.
				{
					0x03, 0x82, 0x62, 0xa6, 0xc6, 0xce,
					0xc9, 0x3c, 0x2d, 0x3e, 0xcd, 0x6c,
					0x60, 0x72, 0xef, 0xea, 0x86, 0xd0,
					0x2f, 0xf8, 0xe3, 0x32, 0x8b, 0xbd,
					0x02, 0x42, 0xb2, 0x0a, 0xf3, 0x42,
					0x59, 0x90,
				},
			},
			class:    WitnessV0PubKeyHashTy,
			pkScript: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			valid := test.pkScript != nil
			pkScript, err := ComputePkScript(
				test.sigScript, test.witness,
			)
			if err != nil && valid {
				t.Fatalf("unable to compute pkScript: %v", err)
			}

			if !valid {
				return
			}

			if pkScript.Class() != test.class {
				t.Fatalf("expected pkScript of type %v, got %v",
					test.class, pkScript.Class())
			}
			if !bytes.Equal(pkScript.Script(), test.pkScript) {
				t.Fatalf("expected pkScript=%x, got pkScript=%x",
					test.pkScript, pkScript.Script())
			}
		})
	}
}
