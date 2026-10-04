import { useLingui } from '@lingui/react/macro'
import { type ReactNode, useEffect, useState } from 'react'
import {
  DeclineOffer,
  Disable,
  Enable,
  Owner,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nxmsvc/service.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { toastError } from '../../toasts/report.ts'
import { useSettings } from '../store.ts'

// 'offer' comes right after the first sign-in; 'takeover' when the switch would take the links from another app.
interface Prompt {
  mode: 'offer' | 'takeover'
  owner: string
}

// Whether Mortar handles nxm links, the switch that changes it and the dialog that asks first.
export function useNxmHandler(): {
  handled: boolean
  owner: string
  toggle: (on: boolean) => void
  offer: () => void
  dialog: ReactNode
} {
  const { t } = useLingui()
  const handled = useSettings((s) => s.nxmHandled)
  const [owner, setOwner] = useState('')
  const [prompt, setPrompt] = useState<Prompt | null>(null)
  useEffect(() => {
    let live = true
    Owner()
      .then((name) => {
        if (live) {
          setOwner(handled ? '' : name)
        }
      })
      .catch(() => {
        if (live) {
          setOwner('')
        }
      })
    return () => {
      live = false
    }
  }, [handled])
  const fail = (e: unknown) => toastError(t`Could not change how Nexus links open`, e)
  const change = (on: boolean) => {
    const done = on ? Enable() : Disable()
    done.catch(fail)
  }
  const ask = (mode: Prompt['mode'], onNone: () => void) => {
    Owner()
      .then((name) =>
        name === '' && mode === 'takeover' ? onNone() : setPrompt({ mode, owner: name }),
      )
      .catch(fail)
  }
  const close = () => setPrompt(null)
  const intro =
    prompt?.mode === 'offer'
      ? t`Should Mortar open Nexus "Mod Manager Download" links, so a click on Nexus starts the download here?`
      : t`Mortar downloads and installs a mod when you click Mod Manager Download on Nexus.`
  const ownerNote = prompt?.owner
    ? t`${prompt.owner} opens these links now. Mortar takes them over and gives them back when you turn this off in Settings.`
    : ''
  const dialog = (
    <ConfirmDialog
      open={prompt !== null}
      title={
        prompt?.mode === 'offer' || !prompt?.owner
          ? t`Handle Nexus download links?`
          : t`Take Nexus links from ${prompt.owner}?`
      }
      body={[intro, ownerNote].filter(Boolean).join(' ')}
      cancelLabel={prompt?.mode === 'offer' ? t`Not now` : t`Cancel`}
      confirmLabel={prompt?.mode === 'offer' ? t`Handle links` : t`Take over`}
      onCancel={() => {
        close()
        if (prompt?.mode === 'offer') {
          DeclineOffer().catch(fail)
        }
      }}
      onConfirm={() => {
        close()
        change(true)
      }}
    />
  )
  return {
    handled,
    owner,
    toggle: (on) => (on ? ask('takeover', () => change(true)) : change(false)),
    offer: () => {
      if (!(useSettings.getState().nxmAsked || useSettings.getState().nxmHandled)) {
        ask('offer', () => undefined)
      }
    },
    dialog,
  }
}
