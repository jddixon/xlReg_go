package reg

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"fmt"
	xc "github.com/jddixon/xlattice_go/crypto"
	xu "github.com/jddixon/xlattice_go/util"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"path"
)

// AAAA makes it run first.
func (s *XLSuite) TestAAAATestDir(c *C) {
	if VERBOSITY > 0 {
		fmt.Println("TEST_TEST_DIR")
	}

	// 001 READ AND INTERPRET test_dir/regCred.dat //////////////////

	rcFile := path.Join("test_dir", "regCred.dat")
	rcData, err := ioutil.ReadFile(rcFile)
	c.Assert(err, IsNil)
	c.Assert(len(rcData) > 0, Equals, true)
	rc, err := ParseRegCred(string(rcData))
	c.Assert(err, IsNil)
	c.Assert(rc, NotNil)

	// 002 HELLO - REPLY TESTS //////////////////////////////////////

	// 201. Read key_rsa as key *rsa.PrivateKey
	keyFile := path.Join("test_dir", "key-rsa")
	kd, err := ioutil.ReadFile(keyFile)
	c.Assert(err, IsNil)
	c.Assert(len(kd) > 0, Equals, true)
	key, err := xc.RSAPrivateKeyFromDisk(kd)
	c.Assert(err, IsNil)
	c.Assert(key, NotNil)

	// 202. Extract public key as pubkey *rsa.PublicKey
	pubKey := key.PublicKey

	// 203. Read key_rsa.pub as pubkey2 *rsa.PublicKey
	pubKeyFile := path.Join("test_dir", "key-rsa.pub")
	pkd, err := ioutil.ReadFile(pubKeyFile)
	c.Assert(err, IsNil)
	c.Assert(len(pkd) > 0, Equals, true)
	pubKey2, err := xc.RSAPubKeyFromDisk(pkd)
	c.Assert(err, IsNil)
	c.Assert(pubKey2, NotNil)

	// 204. Verify pubkey == pubkey2
	c.Assert(&pubKey, DeepEquals, pubKey2)

	// 205. Read version1.str as v1Str
	v1File := path.Join("test_dir", "version1.str")
	v, err := ioutil.ReadFile(v1File)
	c.Assert(err, IsNil)
	c.Assert(len(v) > 0, Equals, true)
	v1Str := string(v)

	// 206. Read version1 as []byte
	v1File = path.Join("test_dir", "version1")
	version1, err := ioutil.ReadFile(v1File)
	c.Assert(err, IsNil)
	c.Assert(len(version1), Equals, 4) // length of DecimalVersion

	// 207. Convert version1 to dv1 DecimalVersion
	dv1, err := xu.VersionFromBytes(version1)
	c.Assert(err, IsNil)

	// 208. Verify v1Str == dv1.String()
	c.Assert(v1Str, Equals, dv1.String())

	// 209-212 same as 205-208 for version2 -------------------------

	// 209. Read version2.str as v2Str
	v2File := path.Join("test_dir", "version2.str")
	v, err = ioutil.ReadFile(v2File)
	c.Assert(err, IsNil)
	c.Assert(len(v) > 0, Equals, true)
	v2Str := string(v)

	// 210. Read version2 as []byte
	v2File = path.Join("test_dir", "version2")
	version2, err := ioutil.ReadFile(v2File)
	c.Assert(err, IsNil)
	c.Assert(len(version2), Equals, 4) // length of DecimalVersion

	// 211. Convert version2 to dv2 DecimalVersion
	dv2, err := xu.VersionFromBytes(version2)
	c.Assert(err, IsNil)

	// 212. Verify v2Str == dv2.String()
	c.Assert(v2Str, Equals, dv2.String())

	// 213-216 read iv1, key1, salt1, hello-data as []byte ----------

	iv1, err := ioutil.ReadFile(path.Join("test_dir", "iv1"))
	c.Assert(err, IsNil)

	key1, err := ioutil.ReadFile(path.Join("test_dir", "key1"))
	c.Assert(err, IsNil)

	salt1, err := ioutil.ReadFile(path.Join("test_dir", "salt1"))
	c.Assert(err, IsNil)

	helloData, err := ioutil.ReadFile(path.Join("test_dir", "hello-data"))
	c.Assert(err, IsNil)

	// 217. helloPlain = iv1 + key1 + salt1 + version1
	var helloPlain []byte
	helloPlain = append(helloPlain, iv1...)
	helloPlain = append(helloPlain, key1...)
	helloPlain = append(helloPlain, salt1...)
	helloPlain = append(helloPlain, version1...)

	// 218. Verify helloPlain == helloData
	bytes.Equal(helloPlain, helloData)

	// 219. Read hello-encrypted as []byte
	helloEncrypted, err := ioutil.ReadFile(
		path.Join("test_dir", "hello-encrypted"))
	c.Assert(err, IsNil)

	// 220. Decrypt helloEncrypted using key => helloDecrypted
	helloDecrypted, err := rsa.DecryptOAEP(sha1.New(), rand.Reader,
		key, helloEncrypted, nil)
	c.Assert(err, IsNil)
	c.Assert(len(helloDecrypted) == 0, Equals, false)

	// 221. Verify helloDecrypted == helloData
	c.Assert(bytes.Equal(helloDecrypted, helloData), Equals, true)

	// 222-226 read iv2, key2, salt2, padding, reply-data as []byte -

	iv2, err := ioutil.ReadFile(path.Join("test_dir", "iv2"))
	c.Assert(err, IsNil)

	key2, err := ioutil.ReadFile(path.Join("test_dir", "key2"))
	c.Assert(err, IsNil)

	salt2, err := ioutil.ReadFile(path.Join("test_dir", "salt2"))
	c.Assert(err, IsNil)

	padding, err := ioutil.ReadFile(path.Join("test_dir", "padding"))
	c.Assert(err, IsNil)

	replyData, err := ioutil.ReadFile(path.Join("test_dir", "reply-data"))
	c.Assert(err, IsNil)

	// 227. helloReply = concat iv2, key2, salt2, version2, salt1, padding
	var helloReply []byte
	helloReply = append(helloReply, iv2...)
	helloReply = append(helloReply, key2...)
	helloReply = append(helloReply, salt2...)
	helloReply = append(helloReply, version2...)
	helloReply = append(helloReply, salt1...)
	helloReply = append(helloReply, padding...)

	// 228. Verify helloReply == replyData
	// DEBUG
	//fmt.Printf("len helloReply %d, len replyData %d\n",
	//	len(helloReply), len(replyData))
	//for i := 0; i < len(helloReply); i++ {
	//	if helloReply[i] != replyData[i] {
	//		fmt.Printf("%02d %02x %02x\n", i, helloReply[i], replyData[i])
	//	}
	//}
	// END
	c.Assert(bytes.Equal(replyData, helloReply), Equals, true)

	// 229. Create aesEngineS1 from iv1, key1
	aesEngineS1, err := aes.NewCipher(key1) // what happens on the server
	c.Assert(err, IsNil)
	c.Assert(aesEngineS1, NotNil)
	aesEncrypter1 := cipher.NewCBCEncrypter(aesEngineS1, iv1)

	// 230. helloReplyMsg = aesEngineS1.encrypt(helloReply)
	helloReplyMsg := make([]byte, len(helloReply))       // includes padding
	aesEncrypter1.CryptBlocks(helloReplyMsg, helloReply) // dest <-src

	// 231. Read reply-encrypted as replyEncrypted []byte
	replyEncrypted, err := ioutil.ReadFile(path.Join("test_dir", "reply-encrypted"))
	c.Assert(err, IsNil)

	// 232. Verify helloReplyMsg == replyEncrypted
	c.Assert(bytes.Equal(helloReplyMsg, replyEncrypted), Equals, true)

	// 233. Create aesEngineC1 from iv1, key1
	aesEngineC1, err := aes.NewCipher(key1) // what happens on the client
	c.Assert(err, IsNil)
	c.Assert(aesEngineC1, NotNil)
	aesDecrypter1c := cipher.NewCBCDecrypter(aesEngineC1, iv1)

	// 234. Use aesEngineC1.decrypt(replyEncrypted) => replyDecrypted
	replyDecrypted := make([]byte, len(replyEncrypted))
	aesDecrypter1c.CryptBlocks(replyDecrypted, replyEncrypted)

	// 235. Verify replyDecrypted == replyData (both with padding)
	c.Assert(bytes.Equal(replyData, replyDecrypted), Equals, true)
}