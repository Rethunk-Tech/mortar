// Joins the folders the way the settings store them: one path-list string, ';' on Windows and ':' elsewhere.
export const separator = navigator.userAgent.includes('Windows') ? ';' : ':'
export const split = (value: string) => (value === '' ? [] : value.split(separator))
