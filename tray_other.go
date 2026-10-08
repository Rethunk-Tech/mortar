//go:build !windows

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// applyTrayMenu refreshes the tray menu in place; GTK and macOS menus track their items.
func applyTrayMenu(_ *application.SystemTray, menu *application.Menu) { menu.Update() }
