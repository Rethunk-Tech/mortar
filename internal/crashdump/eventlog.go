package crashdump

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/nowindow"
)

const (
	eventLogTimeout = 10 * time.Second
	eventLogMaxXML  = 2 << 20
	maxManagedBytes = 16 << 10
	// The provider names and event ids .NET and Windows Error Reporting write an unhandled crash under.
	runtimeProvider  = ".NET Runtime"
	appErrorProvider = "Application Error"
	runtimeEventID   = 1026
	appErrorEventID  = 1000
)

// managedFrame is a frame of a .NET stack in an event's text: "   at Namespace.Type.Method(args)".
var managedFrame = regexp.MustCompile(`^\s+at ([\w.<>` + "`" + `$+]+?)\.[^.\s(]+(?:\[[^\]]*\])? ?\(`)

// frameworkRoots are the root namespaces whose frames are the runtime's or the loader's, never a plugin's.
var frameworkRoots = map[string]bool{"system": true, "microsoft": true, "bepinex": true, "mscorlib": true, "dmd": true, "monomod": true, "harmonyx": true}

// EventLog reads the Windows Application log for a crash of one of exes since `since`: .NET's unhandled-exception
// event (".NET Runtime" 1026) and the native fault event ("Application Error" 1000). It returns nil on other systems,
// where Wine keeps no event log, and when wevtutil is missing, slow or refused.
func EventLog(ctx context.Context, exes []string, since time.Time) []Fault {
	if runtime.GOOS != "windows" {
		return nil
	}
	path, err := exec.LookPath("wevtutil")
	if err != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, eventLogTimeout)
	defer cancel()
	query := fmt.Sprintf("*[System[(Provider[@Name='%s'] or Provider[@Name='%s']) and TimeCreated[@SystemTime>='%s']]]",
		runtimeProvider, appErrorProvider, since.UTC().Format("2006-01-02T15:04:05.000Z"))
	cmd := exec.CommandContext(ctx, path, "qe", "Application", "/q:"+query, "/rd:true", "/c:20", "/f:xml", "/e:Events") // #nosec G204 -- fixed binary and fixed query; only the time is interpolated
	nowindow.Set(cmd)
	out, err := cmd.StdoutPipe()
	if err != nil || cmd.Start() != nil {
		return nil
	}
	raw, _ := io.ReadAll(io.LimitReader(out, eventLogMaxXML))
	cancel()
	_ = cmd.Wait()
	return ParseEventLog(raw, exes)
}

type applicationEvent struct {
	System struct {
		Provider struct {
			Name string `xml:"Name,attr"`
		}
		EventID int
	}
	Data []string `xml:"EventData>Data"`
}

// ParseEventLog turns wevtutil's XML into the faults that name one of exes (case-insensitively). A stream cut off
// by the size cap yields the events before the cut.
func ParseEventLog(raw []byte, exes []string) []Fault {
	var out []Fault
	d := xml.NewDecoder(strings.NewReader(string(raw)))
	for {
		tok, err := d.Token()
		if err != nil {
			return out
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "Event" {
			continue
		}
		var e applicationEvent
		if d.DecodeElement(&e, &start) != nil {
			return out
		}
		if f, ok := e.fault(exes); ok {
			out = append(out, f)
		}
	}
}

func (e applicationEvent) fault(exes []string) (Fault, bool) {
	switch {
	case e.System.Provider.Name == runtimeProvider && e.System.EventID == runtimeEventID && len(e.Data) > 0:
		return runtimeFault(e.Data[0], exes)
	case e.System.Provider.Name == appErrorProvider && e.System.EventID == appErrorEventID && len(e.Data) > 7:
		if !namesExe(e.Data[0], exes) {
			return Fault{}, false
		}
		return Fault{Module: e.Data[3], Detail: fmt.Sprintf("exception %s at offset %s", e.Data[6], e.Data[7])}, true
	}
	return Fault{}, false
}

// runtimeFault reads event 1026's text: an "Application: <exe>" line, then "Exception Info: <type>: <message>" and
// the "   at ..." frames. Frame is the first one outside the runtime and the loader.
func runtimeFault(text string, exes []string) (Fault, bool) {
	var app string
	var managed []string
	inInfo := false
	for line := range strings.SplitSeq(strings.ReplaceAll(text, "\r", ""), "\n") {
		switch {
		case strings.HasPrefix(line, "Application: "):
			app = strings.TrimPrefix(line, "Application: ")
		case strings.HasPrefix(line, "Exception Info: "):
			inInfo = true
			managed = append(managed, strings.TrimPrefix(line, "Exception Info: "))
		case inInfo:
			managed = append(managed, line)
		}
	}
	if !namesExe(app, exes) || len(managed) == 0 {
		return Fault{}, false
	}
	f := Fault{Detail: strings.TrimSpace(managed[0]), Managed: strings.ToValidUTF8(strings.Join(managed, "\n"), "")}
	if len(f.Managed) > maxManagedBytes {
		f.Managed = strings.ToValidUTF8(f.Managed[:maxManagedBytes], "")
	}
	for _, line := range managed[1:] {
		m := managedFrame.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		root, _, _ := strings.Cut(m[1], ".")
		if strings.Contains(m[1], ".") && !frameworkRoots[strings.ToLower(root)] {
			f.Frame = strings.TrimSpace(line)
			break
		}
	}
	return f, true
}

func namesExe(name string, exes []string) bool {
	name = strings.TrimSpace(name)
	for _, exe := range exes {
		if strings.EqualFold(name, exe) {
			return true
		}
	}
	return false
}
