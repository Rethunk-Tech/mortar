import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import { AllowUnknownLayout } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { i18n } from '../i18n/index.ts'
import { useToasts } from '../toasts/store.ts'

/** An archive the game's installer has no layout for, as the install error's `detail` (profile.UnknownLayoutError) carries it. */
export interface UnknownLayout {
  game: string
  key: string
  files: string[]
  /** The player may place it as packed anyway; false when Settings refuses such archives. */
  overridable: boolean
}

const isUnknownLayout = (d: unknown): d is UnknownLayout =>
  typeof d === 'object' &&
  d !== null &&
  typeof (d as Record<string, unknown>).game === 'string' &&
  typeof (d as Record<string, unknown>).key === 'string' &&
  Array.isArray((d as Record<string, unknown>).files) &&
  typeof (d as Record<string, unknown>).overridable === 'boolean'

/** The layout an install stopped on, or null for any other error. */
export function unknownLayoutOf(e: unknown): UnknownLayout | null {
  const cause: unknown = e instanceof Error ? e.cause : undefined
  if (typeof cause !== 'object' || cause === null || !('detail' in cause)) {
    return null
  }
  return isUnknownLayout(cause.detail) ? cause.detail : null
}

/** An install waiting for the player's choice to place an unrecognised archive as packed. */
export interface LayoutAsk {
  title: string
  layout: UnknownLayout
  confirm: () => unknown
}

export const useLayoutAsk = create<{
  asking: LayoutAsk | null
  ask: (a: LayoutAsk) => void
  close: () => void
}>((set) => ({
  asking: null,
  ask: (asking) => set({ asking }),
  close: () => set({ asking: null }),
}))

/** Makes "place as packed" the game's answer from now on, as Vortex's "Install And Don't Ask Again" does. */
export const stopAsking = (game: string) => SetByKey('unknownLayout', 'install', game)

/** The error toast of a refused install; where the player may overrule it the action asks first. */
export function offerUnknownLayout(o: {
  title: string
  name: string
  layout: UnknownLayout
  profile: string
  install: () => Promise<unknown>
}) {
  const { title, name, layout, profile, install } = o
  const body = layout.overridable
    ? i18n._(msg`Mortar has no layout for this mod, so it would place every file as packed.`)
    : i18n._(msg`Mortar has no layout for this mod, and Settings refuses mods like that.`)
  useToasts.getState().push({
    kind: 'error',
    title,
    body,
    ...(layout.overridable
      ? {
          action: {
            label: i18n._(msg`Install anyway…`),
            run: () =>
              useLayoutAsk.getState().ask({
                title: name,
                layout,
                confirm: async () => {
                  await AllowUnknownLayout(layout.game, profile, layout.key)
                  await install()
                },
              }),
          },
        }
      : {}),
  })
}
