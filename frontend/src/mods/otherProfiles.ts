export const selectableProfileIds = (ids: string[], unavailable: Set<string>) =>
  ids.filter((id) => !unavailable.has(id))
