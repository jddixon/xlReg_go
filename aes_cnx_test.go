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

// Make a message (or reply) of up to 16 AES blocks in size and stuff
// it with random bytes.  Return the message with PKCS7-padded appended.
//
func (s *XLSuite) MakeAPaddedMsg(c *C, rng *xr.PRNG) (
	msg []byte, msgLen int, paddedMsg []byte, paddedLen int) {

	msgLen = 2 + rng.Intn(16 * aes.BlockSize - 2)
	msg = make([]byte, msgLen)
	rng.NextBytes(msg)

    // add PKCS7 padding
    paddedMsg, err := xc.AddPKCS7Padding(msg, aes.BlockSize)
    c.Assert(err, IsNil)
    paddedLen = len(paddedMsg)
    nBlks := paddedLen / aes.BlockSize
    c.Assert(paddedLen, Equals, nBlks * aes.BlockSize)	// per contract

	return
}
func (s *XLSuite) encryptMsg(c *C, rng *xr.PRNG, paddedMsg []byte, engine cipher.Block) (
	prefixedCiphertext, iv []byte) {

    // chooose an IV to set up encrypter (later prefix to the padded msg)
	iv = make([]byte, aes.BlockSize)
	rng.NextBytes(iv)
    
    //   A encrypts message
    encrypterA := cipher.NewCBCEncrypter(engine, iv)
    ciphertext := make([]byte, len(paddedMsg))
    encrypterA.CryptBlocks(ciphertext, paddedMsg)	// dest <- src
    
    prefixedCiphertext = make([]byte, len(iv))
    copy(prefixedCiphertext, iv)	// dest <- src
    prefixedCiphertext = append(prefixedCiphertext, ciphertext...)

	return
}
func (s *XLSuite) decryptCiphertext(c *C, rng *xr.PRNG, 
	abCiphertext []byte, engine cipher.Block) (unpaddedMsg , iv []byte) {

	// abCiphertext is prefixed with the (plaintext) IV
	iv = abCiphertext[0:aes.BlockSize]	
	ciphertext := abCiphertext[aes.BlockSize:]
	paddedLen := len(ciphertext)
	c.Assert(ciphertext, DeepEquals, ciphertext)
	decrypterB := cipher.NewCBCDecrypter(engine, iv)
	plaintext := make([]byte, paddedLen)
	decrypterB.CryptBlocks(plaintext, ciphertext)	// dest <- src
	unpaddedMsg, err := xc.StripPKCS7Padding(plaintext, aes.BlockSize)
	c.Assert(err, IsNil)
	return
}
func (s *XLSuite) doTestAESCnx(c *C, rng *xr.PRNG) {

// SESSION SETUP ================================================

	// A->B half circuit ----------------------------------
	keyAB := make([]byte, 2 * aes.BlockSize)
	rng.NextBytes(keyAB)

	// set up A side of A->B half-circuit
	hAOut   := AesCnxHandler{key2: keyAB}
	err := hAOut.SetupSessionKey()
	c.Assert(err, IsNil)
	c.Assert(hAOut.engine, NotNil)

	// set up B side of A->B half-circuit
	hBIn   := AesCnxHandler{key2: keyAB}
	err = hBIn.SetupSessionKey()
	c.Assert(err, IsNil)
	c.Assert(hBIn.engine, NotNil)

	// B->A half circuit ----------------------------------
	keyBA := make([]byte, 2 * aes.BlockSize)
	rng.NextBytes(keyBA)
	
	// set up B side of B->A half-circuit
	hBOut   := AesCnxHandler{key2: keyBA}
	err = hBOut.SetupSessionKey()
	c.Assert(err, IsNil)
	c.Assert(hBOut.engine, NotNil)

	// set up A side of B->A half-circuit
	hAIn   := AesCnxHandler{key2: keyBA}
	err = hAIn.SetupSessionKey()
	c.Assert(err, IsNil)
	c.Assert(hAIn.engine, NotNil)

	// for N messages initiated by A
	N := 4
	for n := 0; n < N; n++ {
		// A create a random-ish message ----------------------------
		msg, msgSize, paddedMsg, paddedLen := s.MakeAPaddedMsg(c, rng)

		// encrypt it, yielding abCiphertext, which is prefixed with the IV
		abCiphertext, ivA := s.encryptMsg(c, rng, paddedMsg, hAOut.engine)

		c.Assert(len(abCiphertext), Equals, paddedLen + aes.BlockSize)

		//   B decrypts msg -----------------------------------------
		unpaddedMsg, ivAb := s.decryptCiphertext(
			c, rng, abCiphertext, hBIn.engine) 
		c.Assert(ivAb, DeepEquals, ivA)
	
		c.Assert(len(unpaddedMsg), Equals, msgSize)
		c.Assert(unpaddedMsg, DeepEquals, msg)

		// B create a random-ish message ----------------------------
		reply, replySize, paddedReply, paddedLen := s.MakeAPaddedMsg(c, rng)

		// encrypt it, yielding baCiphertext, which is prefixed with the IV
		baCiphertext, ivB := s.encryptMsg(c, rng, paddedReply, hBOut.engine)

		c.Assert(len(baCiphertext), Equals, paddedLen + aes.BlockSize)

		//   A decrypts reply -----------------------------------------
		unpaddedReply, ivBb := s.decryptCiphertext(
			c, rng, baCiphertext, hAIn.engine) 
		c.Assert(ivBb, DeepEquals, ivB)
	
		c.Assert(len(unpaddedReply), Equals, replySize)
		c.Assert(unpaddedReply, DeepEquals, reply)
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
