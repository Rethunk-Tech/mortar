package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/packsvc"
)

// profileImport imports another manager's pack: an r2modman code or its key, or a .r2z, profile folder or modpack
// path the app reads itself.
func (c *cmd) profileImport() error {
	a, err := c.need(2, "an r2modman code, key or pack file")
	if err != nil {
		return err
	}
	in := strings.Join(a, " ")
	p := control.Params{Game: c.game, Profile: c.profileFlag, Name: c.nameFlag, Preview: c.previewFlag, Value: in}
	if _, err := os.Stat(in); err == nil {
		if p.Path, err = filepath.Abs(in); err != nil {
			return err
		}
		p.Value = ""
	}
	if c.previewFlag {
		return show(c, "pack.import", p, func(pv packsvc.Preview) {
			fmt.Fprintf(c.out, "%s (%s): %d packages, %d config files\n", pv.Name, pv.Game, len(pv.Packages), pv.Configs)
			for _, pkg := range pv.Packages {
				off := ""
				if pkg.Disabled {
					off = "  (disabled)"
				}
				fmt.Fprintf(c.out, "  %s %s%s\n", pkg.Native, pkg.Version, off)
			}
		})
	}
	var res packsvc.Result
	if err := c.call("pack.import", p, &res, installTimeout); err != nil {
		return err
	}
	return c.emit(res, func() {
		fmt.Fprintf(c.out, "%s\tqueued %d downloads\n", res.Profile, res.Queued)
		if len(res.Unsupported) > 0 {
			fmt.Fprintf(c.out, "Not installable from here: %s\n", strings.Join(res.Unsupported, ", "))
		}
	})
}
