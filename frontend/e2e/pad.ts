import type { Page } from '@playwright/test'

export const A = 0
export const B = 1
export const LB = 4
export const RB = 5
export const DOWN = 13
export const RIGHT = 15

// A standard-mapping pad the page polls through navigator.getGamepads, driven by setting window.pad.
export function installPad(page: Page) {
  return page.addInitScript(() => {
    const state = { buttons: Array.from({ length: 17 }, () => false), axes: [0, 0, 0, 0] }
    Object.assign(globalThis, { pad: state })
    navigator.getGamepads = () => [
      {
        id: 'Steam Virtual Gamepad',
        index: 0,
        connected: true,
        mapping: 'standard',
        timestamp: performance.now(),
        axes: state.axes,
        buttons: state.buttons.map((pressed) => ({
          pressed,
          touched: pressed,
          value: pressed ? 1 : 0,
        })),
        hapticActuators: [],
        vibrationActuator: null,
      } as unknown as Gamepad,
    ]
  })
}

/** Presses and releases one button, a few frames each, as a thumb would. */
export async function press(page: Page, button: number) {
  for (const down of [true, false]) {
    await page.evaluate(
      ([b, d]) =>
        new Promise<void>((done) => {
          ;(globalThis as unknown as { pad: { buttons: boolean[] } }).pad.buttons[b as number] =
            d as boolean
          requestAnimationFrame(() => requestAnimationFrame(() => done()))
        }),
      [button, down] as const,
    )
  }
}
