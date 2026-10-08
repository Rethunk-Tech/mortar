//go:build windows

package lan

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"
)

const (
	fwDirectionIn  = 1
	fwActionBlock  = 0
	fwActionAllow  = 1
	elevatedWaitMS = 5 * 60 * 1000
	ruleName       = "Mortar"
	publicRuleName = "Mortar (public networks)"
)

// policy runs f against the firewall policy object on a locked OS thread, as COM requires.
func policy(f func(policy, rules *ole.IDispatch) error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		var oleErr *ole.OleError
		// S_FALSE: this thread already initialised COM.
		if !errors.As(err, &oleErr) || oleErr.Code() != 1 {
			return fmt.Errorf("initialise COM: %w", err)
		}
	}
	defer ole.CoUninitialize()
	unknown, err := oleutil.CreateObject("HNetCfg.FwPolicy2")
	if err != nil {
		return fmt.Errorf("open Windows Firewall policy: %w", err)
	}
	defer unknown.Release()
	pol, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer pol.Release()
	rulesVar, err := oleutil.GetProperty(pol, "Rules")
	if err != nil {
		return fmt.Errorf("read firewall rules: %w", err)
	}
	defer func() { _ = rulesVar.Clear() }()
	return f(pol, rulesVar.ToIDispatch())
}

func intProp(d *ole.IDispatch, name string) (int32, error) {
	v, err := oleutil.GetProperty(d, name)
	if err != nil {
		return 0, err
	}
	defer func() { _ = v.Clear() }()
	n, ok := v.Value().(int32)
	if !ok {
		return 0, fmt.Errorf("firewall property %s is not a number", name)
	}
	return n, nil
}

func strProp(d *ole.IDispatch, name string) (string, error) {
	v, err := oleutil.GetProperty(d, name)
	if err != nil {
		return "", err
	}
	defer func() { _ = v.Clear() }()
	return v.ToString(), nil
}

func readRule(d *ole.IDispatch) (fwRule, error) {
	app, err := strProp(d, "ApplicationName")
	if err != nil {
		return fwRule{}, err
	}
	dir, err := intProp(d, "Direction")
	if err != nil {
		return fwRule{}, err
	}
	action, err := intProp(d, "Action")
	if err != nil {
		return fwRule{}, err
	}
	enabled, err := oleutil.GetProperty(d, "Enabled")
	if err != nil {
		return fwRule{}, err
	}
	on := enabled.Value() == true
	_ = enabled.Clear()
	profiles, err := intProp(d, "Profiles")
	if err != nil {
		return fwRule{}, err
	}
	return fwRule{App: app, Inbound: dir == fwDirectionIn, Allow: action == fwActionAllow, Enabled: on, Profiles: profiles}, nil
}

// firewallBlocked reports whether Windows Firewall would stop nearby computers reaching Mortar. An unreadable
// firewall is an error, never "not blocked": listening without the rule is what makes Windows raise its own prompt.
func firewallBlocked(_ context.Context, anyAddr bool) (bool, error) {
	exe, err := os.Executable()
	if err != nil {
		return true, fmt.Errorf("find Mortar executable: %w", err)
	}
	var rules []fwRule
	err = policy(func(_, set *ole.IDispatch) error {
		return oleutil.ForEach(set, func(v *ole.VARIANT) error {
			r, err := readRule(v.ToIDispatch())
			if err != nil {
				return err
			}
			rules = append(rules, r)
			return nil
		})
	})
	if err != nil {
		return true, fmt.Errorf("read Windows Firewall rules: %w", err)
	}
	return rulesBlock(rules, exe, anyAddr), nil
}

// networkIsPublic reports whether Windows has the connected network on its Public profile only.
func networkIsPublic() bool {
	var current int32
	err := policy(func(pol, _ *ole.IDispatch) error {
		var err error
		current, err = intProp(pol, "CurrentProfileTypes")
		return err
	})
	return err == nil && current&profilePublic != 0 && current&profilesLocal == 0
}

// AllowFirewall is the elevated step. It removes every inbound block rule for this executable (Windows names the ones
// it creates itself after the file description, not "Mortar") and writes a rule for every network profile, so Windows
// never prompts: allow on Domain and Private with an explicit block on Public, or allow everywhere when anyAddr.
func AllowFirewall(anyAddr bool) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return policy(func(_, set *ole.IDispatch) error {
		// Rules are removed by name, and a name can be shared with another program's rule, so each match is first
		// renamed to a token of its own.
		var tokens []string
		err := oleutil.ForEach(set, func(v *ole.VARIANT) error {
			d := v.ToIDispatch()
			r, err := readRule(d)
			if err != nil {
				return err
			}
			name, err := strProp(d, "Name")
			if err != nil {
				return err
			}
			inboundBlock := r.Inbound && !r.Allow
			if !strings.EqualFold(r.App, exe) || (!inboundBlock && name != ruleName && name != publicRuleName) {
				return nil
			}
			token := "mortar-remove-" + strconv.Itoa(len(tokens))
			if _, err := oleutil.PutProperty(d, "Name", token); err != nil {
				return err
			}
			tokens = append(tokens, token)
			return nil
		})
		if err != nil {
			return err
		}
		for _, token := range tokens {
			if _, err := oleutil.CallMethod(set, "Remove", token); err != nil {
				return fmt.Errorf("remove firewall rule: %w", err)
			}
		}
		if anyAddr {
			return addRule(set, ruleName, exe, fwActionAllow, profilesAll)
		}
		if err := addRule(set, ruleName, exe, fwActionAllow, profilesLocal); err != nil {
			return err
		}
		return addRule(set, publicRuleName, exe, fwActionBlock, profilePublic)
	})
}

func addRule(set *ole.IDispatch, name, exe string, action, profiles int32) error {
	unknown, err := oleutil.CreateObject("HNetCfg.FWRule")
	if err != nil {
		return err
	}
	defer unknown.Release()
	rule, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer rule.Release()
	for prop, value := range map[string]any{
		"Name": name, "ApplicationName": exe, "Direction": int32(fwDirectionIn), "Action": action,
		"Profiles": profiles, "Enabled": true,
	} {
		if _, err := oleutil.PutProperty(rule, prop, value); err != nil {
			return fmt.Errorf("set firewall rule %s: %w", prop, err)
		}
	}
	_, err = oleutil.CallMethod(set, "Add", rule)
	return err
}

// shellExecuteInfo is SHELLEXECUTEINFOW.
type shellExecuteInfo struct {
	Size       uint32
	Mask       uint32
	Hwnd       windows.Handle
	Verb       *uint16
	File       *uint16
	Parameters *uint16
	Directory  *uint16
	Show       int32
	Instance   windows.Handle
	IDList     uintptr
	Class      *uint16
	KeyClass   windows.Handle
	HotKey     uint32
	Icon       windows.Handle
	Process    windows.Handle
}

const seeMaskNoCloseProcess = 0x40

var shellExecuteEx = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteExW")

// fixFirewall runs Mortar itself elevated, so the UAC prompt names Mortar and its publisher, and waits for its exit
// code. Declining the prompt is an error.
func fixFirewall(_ context.Context, anyAddr bool) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find Mortar executable: %w", err)
	}
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	flag := AllowFirewallFlag
	if anyAddr {
		flag = AllowFirewallAny
	}
	args, err := windows.UTF16PtrFromString(flag)
	if err != nil {
		return err
	}
	info := shellExecuteInfo{Mask: seeMaskNoCloseProcess, Verb: verb, File: file, Parameters: args}
	info.Size = uint32(unsafe.Sizeof(info))
	if r, _, callErr := shellExecuteEx.Call(uintptr(unsafe.Pointer(&info))); r == 0 {
		return fmt.Errorf("start the elevated firewall step: %w", callErr)
	}
	defer func() { _ = windows.CloseHandle(info.Process) }()
	if ev, err := windows.WaitForSingleObject(info.Process, elevatedWaitMS); err != nil || ev != windows.WAIT_OBJECT_0 {
		return errors.New("the elevated firewall step did not finish")
	}
	var code uint32
	if err := windows.GetExitCodeProcess(info.Process, &code); err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("update Windows Firewall rule: exit status %d", code)
	}
	return nil
}
