package portal

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	busName    = "org.freedesktop.portal.Desktop"
	objectPath = "/org/freedesktop/portal/desktop"
	launcherIf = "org.freedesktop.portal.DynamicLauncher"
)

// ErrCancelled is returned when the user dismisses the portal's dialog.
var ErrCancelled = errors.New("cancelled in the system dialog")

// requestTimeout bounds the wait for a Response, which may sit behind a dialog the user answers; a portal that never
// answers would otherwise hold the caller forever.
const requestTimeout = 5 * time.Minute

// newToken is a fresh handle token for one call. Each call already has its own connection, and so its own Request
// path; a random token keeps the path from naming the app.
func newToken(kind string) string {
	return "mortar_" + kind + "_" + strings.ToLower(rand.Text())
}

// request calls a portal method that answers through a Request object, and returns the Response's results. It
// subscribes to the Response before calling, since the portal may answer before the call returns.
func request(conn *dbus.Conn, method, token string, args ...any) (map[string]dbus.Variant, error) {
	sender := strings.ReplaceAll(strings.TrimPrefix(conn.Names()[0], ":"), ".", "_")
	handle := dbus.ObjectPath(objectPath + "/request/" + sender + "/" + token)
	match := []dbus.MatchOption{
		dbus.WithMatchObjectPath(handle),
		dbus.WithMatchInterface("org.freedesktop.portal.Request"),
		dbus.WithMatchMember("Response"),
	}
	if err := conn.AddMatchSignal(match...); err != nil {
		return nil, err
	}
	defer func() { _ = conn.RemoveMatchSignal(match...) }()
	signals := make(chan *dbus.Signal, 1)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)
	if err := conn.Object(busName, objectPath).Call(method, 0, args...).Err; err != nil {
		return nil, err
	}
	timeout := time.After(requestTimeout)
	for {
		var sig *dbus.Signal
		select {
		case <-timeout:
			return nil, fmt.Errorf("%s: the portal did not answer", method)
		case s, ok := <-signals:
			if !ok {
				return nil, errors.New("the portal connection closed")
			}
			sig = s
		}
		if sig.Path != handle || len(sig.Body) < 2 {
			continue
		}
		code, _ := sig.Body[0].(uint32)
		results, _ := sig.Body[1].(map[string]dbus.Variant)
		switch code {
		case 0:
			return results, nil
		case 1:
			return nil, ErrCancelled
		default:
			return nil, fmt.Errorf("%s failed", method)
		}
	}
}

// SetAutostart asks the Background portal to start command (run inside the sandbox) at sign-in, or to stop doing so.
func SetAutostart(enable bool, command []string) error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	token := newToken("autostart")
	results, err := request(conn, "org.freedesktop.portal.Background.RequestBackground", token, "",
		map[string]dbus.Variant{
			"handle_token":     dbus.MakeVariant(token),
			"reason":           dbus.MakeVariant("Start Mortar when you sign in"),
			"autostart":        dbus.MakeVariant(enable),
			"commandline":      dbus.MakeVariant(command),
			"dbus-activatable": dbus.MakeVariant(false),
		})
	if err != nil {
		return err
	}
	if granted, _ := results["autostart"].Value().(bool); enable && !granted {
		return errors.New("starting at sign-in was not allowed")
	}
	return nil
}

// InstallLauncher asks the user, through the DynamicLauncher portal, to add a desktop launcher named name that runs
// command inside the sandbox. id must start with the app id and end in .desktop.
func InstallLauncher(id, name, command string, iconPNG []byte) error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	// A serialized GBytesIcon: ("bytes", <PNG data>).
	icon := dbus.MakeVariant(struct {
		Kind string
		Data dbus.Variant
	}{"bytes", dbus.MakeVariant(iconPNG)})
	handleToken := newToken("launcher")
	results, err := request(conn, launcherIf+".PrepareInstall", handleToken, "", name, icon,
		map[string]dbus.Variant{
			"handle_token":  dbus.MakeVariant(handleToken),
			"launcher_type": dbus.MakeVariant(uint32(1)),
		})
	if err != nil {
		return err
	}
	token, _ := results["token"].Value().(string)
	if token == "" {
		return errors.New("the launcher portal returned no install token")
	}
	if chosen, _ := results["name"].Value().(string); chosen != "" {
		name = chosen
	}
	entry := fmt.Sprintf("[Desktop Entry]\nType=Application\nName=%s\nComment=Play this Mortar profile\nExec=%s\nTerminal=false\nCategories=Game;\n",
		desktopEntryText(name), command)
	return conn.Object(busName, objectPath).Call(launcherIf+".Install", 0, token, id, entry,
		map[string]dbus.Variant{}).Err
}

// UninstallLauncher removes a launcher InstallLauncher added; a missing one is not an error.
func UninstallLauncher(id string) error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	if exists, err := launcherExists(conn, id); err != nil || !exists {
		return err
	}
	return conn.Object(busName, objectPath).Call(launcherIf+".Uninstall", 0, id, map[string]dbus.Variant{}).Err
}

// LauncherExists reports whether InstallLauncher added the launcher id.
func LauncherExists(id string) (bool, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return false, err
	}
	defer func() { _ = conn.Close() }()
	return launcherExists(conn, id)
}

func launcherExists(conn *dbus.Conn, id string) (bool, error) {
	var entry string
	err := conn.Object(busName, objectPath).Call(launcherIf+".GetDesktopEntry", 0, id).Store(&entry)
	if err == nil {
		return true, nil
	}
	// The portal reads the launcher from its own folder and reports a missing one as GLib's G_FILE_ERROR_NOENT (4).
	var dbusErr dbus.Error
	if errors.As(err, &dbusErr) && strings.HasSuffix(dbusErr.Name, "_g_2dfile_2derror_2dquark.Code4") {
		return false, nil
	}
	return false, err
}

// RetrieveSecret asks the Secret portal for this app's master secret, the same bytes on every call, which the portal
// writes to the pipe it is given.
func RetrieveSecret() ([]byte, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	fd := w.Fd()
	if fd > math.MaxInt32 {
		_ = w.Close()
		return nil, errors.New("pipe descriptor out of range")
	}
	token := newToken("secret")
	_, err = request(conn, "org.freedesktop.portal.Secret.RetrieveSecret", token,
		dbus.UnixFD(fd), map[string]dbus.Variant{"handle_token": dbus.MakeVariant(token)})
	// Our copy of the write end must close or the read never sees EOF.
	_ = w.Close()
	if err != nil {
		return nil, err
	}
	secret, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(secret) == 0 {
		return nil, errors.New("the secret portal returned an empty secret")
	}
	return secret, nil
}

// desktopEntryText is name as a Desktop Entry string value: a backslash starts an escape there, so it is doubled,
// and a line break would end the key, so it becomes a space.
func desktopEntryText(name string) string {
	return strings.NewReplacer("\\", `\\`, "\n", " ", "\r", " ").Replace(name)
}
