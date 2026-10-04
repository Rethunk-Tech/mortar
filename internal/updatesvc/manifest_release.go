//go:build !updatetest

package updatesvc

// serverUpdates is false in every shipped build: a server-mode build never updates itself.
const serverUpdates = false

func manifestURL() string { return ManifestURL }
