//go:build !server

package backdrop

// negotiates is false in the desktop window: WebKitGTK decodes JPEG XL without listing it in Accept.
const negotiates = false
