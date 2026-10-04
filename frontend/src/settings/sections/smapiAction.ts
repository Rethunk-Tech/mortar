interface SmapiAction {
  kind: 'install' | 'update' | 'reinstall'
  version: string
  confirm: boolean
}

interface SmapiState {
  installed: boolean
  broken: boolean
  version: string
  updateAvailable: boolean
}

/** The one action the SMAPI row offers for the chosen version: following latest installs or updates at once, while
 * a pinned other version asks first because it changes what every profile runs. */
function smapiAction(status: SmapiState, selected: string, latestValue: string): SmapiAction {
  const pinned = selected !== latestValue
  if (!status.installed) {
    return { kind: 'install', version: pinned ? selected : '', confirm: pinned }
  }
  if (status.broken) {
    return { kind: 'reinstall', version: pinned ? selected : '', confirm: false }
  }
  if (!pinned) {
    return { kind: status.updateAvailable ? 'update' : 'reinstall', version: '', confirm: false }
  }
  if (selected === status.version) {
    return { kind: 'reinstall', version: selected, confirm: false }
  }
  return { kind: 'install', version: selected, confirm: true }
}

export { type SmapiAction, smapiAction }
