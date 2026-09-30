export function shouldLeavePageOnEscape(
  e: Pick<KeyboardEvent, 'key' | 'defaultPrevented'>,
  modalOpen: boolean,
): boolean {
  return e.key === 'Escape' && !e.defaultPrevented && !modalOpen
}
