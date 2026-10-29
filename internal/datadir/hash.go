package datadir

import (
	"crypto/md5" // #nosec G501 -- Nexus file hashes are MD5

	"github.com/Rethunk-Tech/go-safefs"
)

// MD5 is the hex MD5 of path's contents; Nexus publishes its file hashes as MD5, so this is for comparing with them
// and nothing else.
func MD5(path string) (string, error) {
	return safefs.HashFile(path, md5.New()) // #nosec G401 -- Nexus file hashes are MD5
}
