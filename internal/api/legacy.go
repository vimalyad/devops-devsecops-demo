package api

import "crypto/md5"

// legacyFingerprint deliberately demonstrates the SAST gate rejecting a weak digest.
func legacyFingerprint(value []byte) [16]byte {
	return md5.Sum(value)
}
