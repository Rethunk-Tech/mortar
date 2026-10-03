import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from '@mui/material'
import { type ReactNode, useEffect, useState } from 'react'
import {
  DeclineOffer,
  Disable,
  Enable,
  Owner,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/nxmsvc/service.ts'
import { paper } from '../../mods/paper.ts'
import { errorDetails, errorMessage } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
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
  const fail = (e: unknown) =>
    useToasts.getState().push({
      kind: 'error',
      title: t`Could not change how Nexus links open`,
      body: errorMessage(e),
      detail: errorDetails(e),
    })
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
    <Dialog open={prompt !== null} onClose={close} transitionDuration={0} slotProps={{ paper }}>
      <DialogTitle>
        {prompt?.mode === 'offer' || !prompt?.owner
          ? t`Handle Nexus download links?`
          : t`Take Nexus links from ${prompt.owner}?`}
      </DialogTitle>
      <DialogContent>
        <DialogContentText>{[intro, ownerNote].filter(Boolean).join(' ')}</DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button
          onClick={() => {
            close()
            if (prompt?.mode === 'offer') {
              DeclineOffer().catch(fail)
            }
          }}
        >
          {prompt?.mode === 'offer' ? t`Not now` : t`Cancel`}
        </Button>
        <Button
          variant="contained"
          onClick={() => {
            close()
            change(true)
          }}
        >
          {prompt?.mode === 'offer' ? t`Handle links` : t`Take over`}
        </Button>
      </DialogActions>
    </Dialog>
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
