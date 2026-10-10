// A fragment with its first letter raised, to stand as a sentence or a label.
export const sentence = (text: string): string => text.charAt(0).toUpperCase() + text.slice(1)
