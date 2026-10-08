package main

import "github.com/wailsapp/wails/v3/pkg/application"

// applyTrayMenu hands the filled menu to the tray again: Windows builds the right-click popup from the items the menu
// holds when it is set, and Menu.Update never rebuilds it, so a menu filled after SetMenu would open empty.
func applyTrayMenu(tray *application.SystemTray, menu *application.Menu) { tray.SetMenu(menu) }
