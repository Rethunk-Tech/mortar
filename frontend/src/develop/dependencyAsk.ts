import { msg } from '@lingui/core/macro'
import { create } from 'zustand'
import { AddDependency } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { i18n } from '../i18n/index.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

/** A Browse pick waiting for the player to say how the project depends on it. */
export interface DependencyAsk {
  game: string
  dir: string
  project: string
  pkg: string
  name: string
  version: string
  // Installs the package into the development profile; null when it is already there.
  install: (() => void) | null
}

export const useDependencyAsk = create<{ ask: DependencyAsk | null }>(() => ({ ask: null }))

/** Makes the package a dependency of the project the ask names, installing it first when it is not there yet. */
export function addDependency(ask: DependencyAsk, soft: boolean) {
  ask.install?.()
  AddDependency(ask.game, ask.dir, ask.pkg, ask.version, soft)
    .then(() =>
      useToasts.getState().push({
        kind: 'success',
        title: soft
          ? i18n._(msg`${{ mod: ask.name }} is an optional dependency of ${{ name: ask.project }}`)
          : i18n._(msg`${{ mod: ask.name }} is a dependency of ${{ name: ask.project }}`),
      }),
    )
    .catch(reportError(i18n._(msg`Could not add the dependency`)))
}
