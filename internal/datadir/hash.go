package datadir

import (
	"crypto/md5"  //nolint:gosec // Nexus file hashes are MD5
	"crypto/sha1" //nolint:gosec // CurseForge file hashes are SHA-1

	"github.com/Rethunk-Tech/go-safefs"
)

// MD5 is the hex MD5 of path's contents; Nexus publishes its file hashes as MD5, so this is for comparing with them
// and nothing else.
func MD5(path string) (string, error) {
	return safefs.HashFile(path, md5.New()) //nolint:gosec // Nexus file hashes are MD5
}

// SHA1 is the hex SHA-1 of path's contents; CurseForge publishes its file hashes as SHA-1, so this is for comparing
// with them and nothing else.
func SHA1(path string) (string, error) {
	return safefs.HashFile(path, sha1.New()) //nolint:gosec // CurseForge file hashes are SHA-1
}
