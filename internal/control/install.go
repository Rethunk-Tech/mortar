package control

import "fmt"

// needNoInstall refuses an install-targeted verb whose launch service entry point is game-wide.
func needNoInstall(p Params, verb string) error {
	if p.Install == "" {
		return nil
	}
	return fmt.Errorf("%s cannot target install %q: the launch service has no install-targeted %s", verb, p.Install, verb)
}
