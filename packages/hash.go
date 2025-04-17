package packages

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Argon2ID struct {
	Format  string
	Version int
	Time    uint32
	Memory  uint32
	KeyLen  uint32
	SaltLen uint32
	Threads uint8
}

func (arg *Argon2ID) NewArgon2ID() {
	arg.Format = "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s"
	arg.Version = argon2.Version
	arg.Time = 1
	arg.Memory = 64 * 1024
	arg.KeyLen = 32
	arg.SaltLen = 16
	arg.Threads = 4
}

func (arg *Argon2ID) Hash(plain string) (string, error) {
	salt := make([]byte, arg.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(plain), salt, arg.Time, arg.Memory, arg.Threads, arg.KeyLen)

	return fmt.Sprintf(
			arg.Format,
			arg.Version,
			arg.Memory,
			arg.Time,
			arg.Threads,
			base64.RawStdEncoding.EncodeToString(salt),
			base64.RawStdEncoding.EncodeToString(hash),
		),
		nil
}

func (arg *Argon2ID) Verify(plain, hash string) (bool, error) {
	hashParts := strings.Split(hash, "$")

	_, err := fmt.Sscanf(hashParts[3], "m=%d,t=%d,p=%d", &arg.Memory, &arg.Time, &arg.Threads)
	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(hashParts[4])
	if err != nil {
		return false, err
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(hashParts[5])
	if err != nil {
		return false, err
	}

	hashToCompare := argon2.IDKey([]byte(plain), salt, arg.Time, arg.Memory, arg.Threads, uint32(len(decodedHash)))

	return subtle.ConstantTimeCompare(decodedHash, hashToCompare) == 1, nil
}
