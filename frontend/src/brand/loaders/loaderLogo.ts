import bepinex from '../vendor/bepinex.png'
import melonloader from '../vendor/melonloader.png'
import smapi from '../vendor/smapi.png'

// Each known loader's own mark, shipped with Mortar; a loader without one shows a plain mark.
const LOGOS: Record<string, string> = { bepinex5: bepinex, melonloader, smapi }

/** loaderLogo is the address of the loader's mark, '' when Mortar has none for it. */
export function loaderLogo(id: string): string {
  return LOGOS[id] ?? ''
}
