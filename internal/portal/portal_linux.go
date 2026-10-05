package portal

import (
	"errors"
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	busName    = "org.freedesktop.portal.Desktop"
	objectPath = "/org/freedesktop/portal/desktop"
	launcherIf = "org.freedesktop.portal.DynamicLauncher"
)

// ErrCancelled is returned when the user dismisses the portal's dialog.
var ErrCancelled = errors.New("cancelled in the system dialog")

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
	for sig := range signals {
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
	return nil, errors.New("the portal connection closed")
}

// SetAutostart asks the Background portal to start command (run inside the sandbox) at sign-in, or to stop doing so.
func SetAutostart(enable bool, command []string) error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	results, err := request(conn, "org.freedesktop.portal.Background.RequestBackground", "mortar_autostart", "",
		map[string]dbus.Variant{
			"handle_token":     dbus.MakeVariant("mortar_autostart"),
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
	results, err := request(conn, launcherIf+".PrepareInstall", "mortar_launcher", "", name, icon,
		map[string]dbus.Variant{
			"handle_token":  dbus.MakeVariant("mortar_launcher"),
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
		strings.NewReplacer("\n", " ", "\r", " ").Replace(name), command)
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
