package nativehost

// Start runs Mortar at exe on link without waiting: the new process forwards the link to the running Mortar, or
// becomes it.
func Start(exe, link string) error { return start(exe, link) }
