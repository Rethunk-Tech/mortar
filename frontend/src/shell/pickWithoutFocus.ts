// A click picks a menu item without focusing it first: focus would move the menu's active item, and MUI redraws every
// row for that, just before the menu closes. The item the menu focused as it opened is blurred too, so a dialog the
// pick opens does not inherit its keyboard focus ring. Keyboard focus is unaffected.
export const pickWithoutFocus = (e: { preventDefault: () => void }) => {
  e.preventDefault()
  if (document.activeElement instanceof HTMLElement) {
    document.activeElement.blur()
  }
}
