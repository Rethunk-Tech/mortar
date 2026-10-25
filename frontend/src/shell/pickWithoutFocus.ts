// A click picks a menu item without focusing it first: focus would move the menu's active item, and MUI redraws every
// row for that, just before the menu closes. Keyboard focus is unaffected.
export const pickWithoutFocus = (e: { preventDefault: () => void }) => e.preventDefault()
