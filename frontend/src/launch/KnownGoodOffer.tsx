import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Button, Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material'
import { useEffect, useRef, useState } from 'react'
import { State } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { Runs } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import { MarkKnownGood } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useLaunch } from './store.ts'

export function KnownGoodOffer() {
  const { t } = useLingui()
  const status = useLaunch((s) => s.status)
  const prev = useRef(status)
  const [offer, setOffer] = useState<{ game: string; profile: string } | null>(null)
  useEffect(() => {
    const was = prev.current
    prev.current = status
    if (
      status?.state === State.Idle &&
      was?.state === State.Running &&
      status.game !== '' &&
      status.profile !== ''
    ) {
      const { game, profile } = status
      Runs(game, profile)
        .then((list) => {
          const run = list?.[0]
          if (run && run.outcome === 'ran' && (run.errors ?? 0) === 0) {
            setOffer({ game, profile })
          }
        })
        .catch(() => undefined)
    }
  }, [status])
  if (!offer) {
    return null
  }
  return (
    <Dialog open={true} onClose={() => setOffer(null)} transitionDuration={0}>
      <DialogTitle>{t`Mark this profile known good?`}</DialogTitle>
      <DialogContent>{t`That run reached the title screen without errors.`}</DialogContent>
      <DialogActions>
        <Button onClick={() => setOffer(null)} sx={{ whiteSpace: 'nowrap' }}>
          {t`Not now`}
        </Button>
        <Button
          onClick={() => {
            MarkKnownGood(offer.game, offer.profile)
              .then(() => {
                useToasts.getState().push({
                  kind: 'success',
                  title: i18n._(msg`Marked known good`),
                })
                setOffer(null)
              })
              .catch(reportUnexpected)
          }}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t`Mark known good`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
