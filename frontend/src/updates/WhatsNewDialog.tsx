import { useLingui } from '@lingui/react/macro'
import {
  Box,
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
import type { ReleaseNotes as ReleaseNotesResult } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/updatesvc/models.ts'
import {
  AckWhatsNew,
  WhatsNew as LoadWhatsNew,
  ReleaseNotes,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/updatesvc/service.ts'
import { openPage } from '../mods/menu.ts'
import { PageLink } from '../share/CollectionNotes.tsx'
import { parseInstructions } from '../share/instructions.ts'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { showWhatsNew, useWhatsNew } from './whatsNew.ts'

const HEADING = /^#+\s*/

const BULLET = /^[-*]\s+/

function NoteParts({ line }: { line: ReturnType<typeof parseInstructions>[number] }) {
  const first = line.parts[0]?.text ?? ''
  const strip = [HEADING, BULLET].find((marker) => marker.test(first))
  return line.parts.map((part, i) => (
    <PageLink
      key={part.at}
      part={i === 0 && strip ? { ...part, text: part.text.replace(strip, '') } : part}
    />
  ))
}

type Line = ReturnType<typeof parseInstructions>[number]

const isBullet = (line: Line) => BULLET.test(line.parts[0]?.text ?? '')

function NoteLine({ line }: { line: Line }) {
  const heading = HEADING.test(line.parts[0]?.text ?? '')
  const bullet = isBullet(line)
  let component: 'h3' | 'li' | 'p' = 'p'
  if (heading) {
    component = 'h3'
  } else if (bullet) {
    component = 'li'
  }
  return (
    <Typography
      component={component}
      variant={heading ? 'subtitle2' : 'body2'}
      sx={{ m: 0, minHeight: '1.4em', color: heading ? 'text.primary' : 'var(--mortar-ink-sec)' }}
    >
      <NoteParts line={line} />
    </Typography>
  )
}

// Consecutive bullet lines share one list; every other line stands alone.
function Notes({ text }: { text: string }) {
  const groups: Line[][] = []
  for (const line of parseInstructions(text)) {
    const last = groups.at(-1)
    if (last && isBullet(line) && isBullet(last[0] as Line)) {
      last.push(line)
    } else {
      groups.push([line])
    }
  }
  return groups.map((group) =>
    isBullet(group[0] as Line) ? (
      <Box key={group[0]?.at} component="ul" sx={{ m: 0, pl: space.pad }}>
        {group.map((line) => (
          <NoteLine key={line.at} line={line} />
        ))}
      </Box>
    ) : (
      <NoteLine key={group[0]?.at} line={group[0] as Line} />
    ),
  )
}

export function WhatsNewDialog() {
  const { t } = useLingui()
  const shown = useWhatsNew()
  const [release, setRelease] = useState<ReleaseNotesResult | null>(null)
  const version = shown?.version
  useEffect(() => {
    LoadWhatsNew()
      .then((w) => (w.notes && w.version ? showWhatsNew(w.version, w.notes) : undefined))
      .catch(reportUnexpected)
  }, [])
  useEffect(() => {
    setRelease(null)
    if (version) {
      ReleaseNotes(version).then(setRelease).catch(reportUnexpected)
    }
  }, [version])
  const close = () => {
    useWhatsNew.setState(null, true)
    AckWhatsNew().catch(reportUnexpected)
  }
  return (
    <Dialog open={shown !== null} onClose={close} slotProps={{ paper: { sx: { maxWidth: 520 } } }}>
      <DialogTitle>{t`What's new in Mortar ${version ?? ''}`}</DialogTitle>
      <DialogContent>
        {release?.available ? (
          <Notes text={release.notes} />
        ) : (
          <DialogContentText
            component="div"
            sx={{ whiteSpace: 'pre-wrap', fontSize: 14, color: 'var(--mortar-ink-sec)' }}
          >
            {shown?.fallback}
          </DialogContentText>
        )}
        {release && !release.available ? (
          <Link
            component="button"
            type="button"
            sx={{ mt: 1, fontSize: 13 }}
            onClick={() =>
              openPage(`https://github.com/Rethunk-Tech/mortar/releases/tag/v${version}`)
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
