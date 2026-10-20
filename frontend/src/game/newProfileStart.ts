// A starting point in the New profile dialog's "Start from" list. A saved group's value carries its id after this
// prefix; the control character keeps it apart from a template name, which cannot start with one.
const GROUP = '\u0001group:'

export const groupStart = (id: string): string => GROUP + id

/** The saved group id a "Start from" value names, or null for any other start. */
export function savedGroupOf(value: string): string | null {
  return value.startsWith(GROUP) && value.length > GROUP.length ? value.slice(GROUP.length) : null
}
