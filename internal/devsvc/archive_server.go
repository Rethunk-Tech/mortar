//go:build server

package devsvc

import "os"

// Self-test servers can take the template from MORTAR_DEV_TEMPLATE_URL, so a template not yet published can be tried.
func archiveOverride() string { return os.Getenv("MORTAR_DEV_TEMPLATE_URL") }
