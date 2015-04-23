package reg

// xlReg_go/aes_cnx_test.go

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	xr "github.com/jddixon/rnglib_go"
	xc "github.com/jddixon/xlCrypto_go"
	//xe "github.com/jddixon/xlProtocol_go/aes_cnx"
	. "gopkg.in/check.v1"
)

func (s *XLSuite) doTestAESCnx(c *C, rng *xr.PRNG) {

	// SESSION SETUP ================================================
	// set up A side of session
	keyA := make([]byte, 2 * aes.BlockSize)
	rng.NextBytes(keyA)
	hA   := AesCnxHandler{key2: keyA}

	err := hA.SetupSessionKey()
	c.Assert(err, IsNil)
	c.Assert(hA.engine, NotNil)

	// set up B side of session
	keyB := make([]byte, 2 * aes.BlockSize)
	rng.NextBytes(keyB)
	hB   := AesCnxHandler{key2: keyB}

	err = hB.SetupSessionKey()
	c.Assert(err, IsNil)
	c.Assert(hB.engine, NotNil)

	// for N messages initiated by A
	N := 1
	for n := 0; n < N; n++ {
		// A SENDS MESSAGE TO B =====================================
		
		// A create a random-ish message ----------------------------
		msgASize := 2 + rng.Intn(16 * aes.BlockSize - 2) 
		msgA := make([]byte, msgASize)
		rng.NextBytes(msgA)

		// A adds PKCS7 padding
		paddedMsg, err := xc.AddPKCS7Padding(msgA, aes.BlockSize)
		c.Assert(err, IsNil)
		paddedLen := len(paddedMsg)
		// DEBUG
		fmt.Printf("msgLen %d, padded %d\n", msgASize, paddedLen)
		// END
		nBlks := paddedLen / aes.BlockSize
		c.Assert(paddedLen, Equals, nBlks * aes.BlockSize)	// per contract

		// chooose an IV to set up encrypter (later prefix to the padded msg)
		ivA := make([]byte, aes.BlockSize)
		rng.NextBytes(ivA)

		//   A encrypts message
		encrypterA := cipher.NewCBCEncrypter(hA.engine, ivA)
		ciphertext := make([]byte, paddedLen)
		encrypterA.CryptBlocks(ciphertext, paddedMsg)	// dest <- src

		prefixedCiphertext := make([]byte, len(ivA))
		copy(prefixedCiphertext, ivA)	// dest <- src
		prefixedCiphertext = append(prefixedCiphertext, paddedMsg...)
		// DEBUG
		lenPrefixed := len(prefixedCiphertext)
		// END
		c.Assert(len(prefixedCiphertext), Equals, (nBlks + 1) * aes.BlockSize)

		//   B decrypts msg -----------------------------------------
		ivAb := prefixedCiphertext[0:aes.BlockSize]	// extract the IV
		c.Assert(ivAb,DeepEquals,ivA)
		bCiphertextIn := prefixedCiphertext[aes.BlockSize:]
		lenCipherIntoB := len(bCiphertextIn)
		c.Assert(lenCipherIntoB/aes.BlockSize, Equals, nBlks)
		decrypterB := cipher.NewCBCEncrypter(hB.engine, ivAb)
		plaintext := make([]byte, lenCipherIntoB)
		decrypterB.CryptBlocks(plaintext, bCiphertextIn)	// dest <- src
		unpaddedMsg, err := xc.StripPKCS7Padding(plaintext, aes.BlockSize)
		c.Assert(err, IsNil)
		// DEBUG
		fmt.Printf("B-side prefixed ciphertext %d bytes, plaintext %d, stripped %d\n",
			lenPrefixed, len(plaintext), len(unpaddedMsg))
		// END
		c.Assert(len(unpaddedMsg), Equals, msgASize)
		c.Assert(unpaddedMsg, Equals, msgA)

		// B SENDS REPLY TO A =======================================

		// B create a random-ish reply
		replyBSize := 2 + rng.Intn(16 * aes.BlockSize - 2) 
		replyB := make([]byte, replyBSize)
		rng.NextBytes(replyB)

		//   B encrypts reply

		//   A decrypts reply 
	}
}

func (s *XLSuite) TestAESCnx(c *C) {
	if VERBOSITY > 0 {
		fmt.Println("\nTEST_AES_CNX")
	}
	rng := xr.MakeSimpleRNG()

	K := 1	// XXX 
	for k := 0; k < K; k++ {
		s.doTestAESCnx(c, rng)
	}
	//c.Assert(err, IsNil)
	//c.Assert(cm.Equal(cm2), Equals, true)
}
