// A label reused mid-sentence starts lowercase there, so the label stays one message (docs/gui-design.md, Copy).
export function lowerFirst(text: string): string {
  return text.charAt(0).toLocaleLowerCase() + text.slice(1)
}
