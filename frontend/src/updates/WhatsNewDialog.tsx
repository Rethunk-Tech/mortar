import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Link,
  Typography,
} from '@mui/material'
import { useEffect, useState } from 'react'
import type {
  ReleaseNotes as ReleaseNotesResult,
  WhatsNew,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/updatesvc/models.ts'
import {
  AckWhatsNew,
  WhatsNew as LoadWhatsNew,
  ReleaseNotes,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/updatesvc/service.ts'
import { openPage } from '../mods/menu.ts'
import { PageLink } from '../share/CollectionNotes.tsx'
import { parseInstructions } from '../share/instructions.ts'

const HEADING = /^#+\s*/

function Notes({ text }: { text: string }) {
  return parseInstructions(text).map((line) => {
    const first = line.parts[0]?.text ?? ''
    const heading = HEADING.test(first)
    return (
      <Typography
        key={line.at}
        variant={heading ? 'subtitle2' : 'body2'}
        sx={{ minHeight: '1.4em', color: heading ? 'text.primary' : 'var(--mortar-ink-sec)' }}
      >
        {line.parts.map((part, i) => (
          <PageLink
            key={part.at}
            part={i === 0 && heading ? { ...part, text: part.text.replace(HEADING, '') } : part}
          />
        ))}
      </Typography>
    )
  })
}

export function WhatsNewDialog() {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  const [payload, setPayload] = useState<WhatsNew | null>(null)
  const [release, setRelease] = useState<ReleaseNotesResult | null>(null)
  useEffect(() => {
    LoadWhatsNew()
      .then((w) => {
        if (!(w.notes && w.version)) {
          return null
        }
        setPayload(w)
        setOpen(true)
        return ReleaseNotes(w.version)
      })
      .then(setRelease)
      .catch(() => undefined)
  }, [])
  const close = () => {
    setOpen(false)
    AckWhatsNew().catch(() => undefined)
  }
  return (
    <Dialog open={open} onClose={close} slotProps={{ paper: { sx: { maxWidth: 520 } } }}>
      <DialogTitle>{t`What's new in Mortar ${payload?.version ?? ''}`}</DialogTitle>
      <DialogContent>
        {release?.available ? (
          <Notes text={release.notes} />
        ) : (
          <DialogContentText
            component="div"
            sx={{ whiteSpace: 'pre-wrap', fontSize: 14, color: 'var(--mortar-ink-sec)' }}
          >
            {payload?.notes}
          </DialogContentText>
        )}
        {release && !release.available ? (
          <Link
            component="button"
            type="button"
            sx={{ mt: 1, fontSize: 13 }}
            onClick={() =>
              openPage(`https://github.com/Rethunk-AI/mortar/releases/tag/v${payload?.version}`)
            }
          >
            {t`Read the release notes on GitHub`}
          </Link>
        ) : null}
      </DialogContent>
      <DialogActions>
        <Button variant="contained" onClick={close}>
          {t`Got it`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
