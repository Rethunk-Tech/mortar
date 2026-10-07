package cli

import "github.com/Rethunk-Tech/mortar/internal/usererr"

// Sentence is the user-facing line for a kind. Same English as the GUI errorMessage mapper (frontend/src/toasts/report.ts), which a test compares.
func Sentence(kind usererr.Kind) string {
	switch kind {
	case usererr.NotFound:
		return "That item could not be found."
	case usererr.Busy:
		return "The game is already running."
	case usererr.Network:
		return "A network request failed."
	case usererr.Permission:
		return "Mortar does not have permission to do that."
	case usererr.DiskFull:
		return "The disk is full."
	case usererr.Damaged:
		return "That data could not be read."
	case usererr.Invalid:
		return "That request was not valid."
	case usererr.OtherGame:
		return "That is for another game. Open that game and import it there."
	case usererr.Outdated:
		return "That was made by a newer Mortar. Update Mortar to open it."
	case usererr.External:
		return "The author only allows this download on the source's own page. Download the file there and add it from your computer."
	case usererr.Malware:
		return "The antivirus flagged this download, so Mortar did not install it."
	case usererr.Locked:
		return "Waiting for you to unlock the keyring."
	case usererr.Unknown:
		fallthrough
	default:
		return "Something went wrong."
	}
}
