import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Radio,
  RadioGroup,
  Typography,
} from '@mui/material'
import { useState } from 'react'
import type {
  Copy,
  Duplicate,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { nexusKeepKey, preselect } from './lookup.ts'
import { LetterTile } from './parts.tsx'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

function CopyOption({ copy, dup, differ }: { copy: Copy; dup: Duplicate; differ: boolean }) {
  const { t } = useLingui()
  const source = (() => {
    if (copy.nexus) {
      return t`From Nexus`
    }
    return copy.source === 'smapi' ? t`Bundled with SMAPI` : t`From an archive`
  })()
  const needed = copy.needed ?? []
  const tooOld = copy.tooOld ?? []
  const lines = [
    differ && copy.newest ? t`Newer.` : '',
    differ && !copy.newest ? t`Older.` : '',
    copy.nexus ? t`Gets update checks.` : t`No update checks, and it can't go in a share link.`,
    needed.length > 0 ? t`${needed.join(', ')} needs this version or newer.` : '',
    tooOld.length > 0 ? t`${tooOld.join(', ')} needs a newer version than this.` : '',
  ].filter(Boolean)
  return (
    <Box
      component="label"
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: 1.25,
        p: 2,
        cursor: 'pointer',
        bgcolor: 'var(--mortar-raised)',
        border: '2px solid transparent',
        borderRadius: '8px',
        '&:has(input:checked)': { borderColor: 'primary.main' },
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25 }}>
        <Radio
          value={copy.key}
          sx={{ p: 0 }}
          slotProps={{ input: { 'aria-label': t`${source}, version ${copy.version}` } }}
        />
        <LetterTile mod={{ uniqueId: dup.uniqueId, name: copy.name }} size={40} />
        <Box sx={{ minWidth: 0 }}>
          <Typography sx={{ fontSize: 15, fontWeight: 700 }}>{source}</Typography>
          <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
            {t`Version ${copy.version}`}
          </Typography>
        </Box>
      </Box>
      <Typography sx={{ fontSize: 13, lineHeight: 1.5 }}>{lines.join(' ')}</Typography>
    </Box>
  )
}

function Resolver({ dup, profileName }: { dup: Duplicate; profileName: string }) {
  const { t } = useLingui()
  const resolve = useMods((s) => s.resolve)
  const keepCopy = useMods((s) => s.keepCopy)
  const locked = useLocked()
  const lockedTitle = t`Stop the game to change mods.`
  const copies = dup.copies ?? []
  const [keep, setKeep] = useState(preselect(copies))
  const nexusKey = nexusKeepKey(copies)
  const differ = copies.some((c) => !c.newest)
  return (
    <>
      <DialogTitle sx={{ fontSize: 22, fontWeight: 700 }}>
        {t`${copies.length} copies of ${dup.name}`}
      </DialogTitle>
      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        <Typography sx={{ fontSize: 14, lineHeight: 1.5 }}>
          {t`${profileName} has ${dup.name} more than once. SMAPI loads only one, so pick which to keep. The others are switched off, not deleted.`}
        </Typography>
        <RadioGroup
          aria-label={t`Copy to keep`}
          value={keep}
          onChange={(e) => setKeep(e.target.value)}
          sx={{
            display: 'grid',
            gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
            gap: 1.5,
          }}
        >
          {copies.map((c) => (
            <CopyOption key={c.key} copy={c} dup={dup} differ={differ} />
          ))}
        </RadioGroup>
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 2.5 }}>
        <Button onClick={() => resolve(null)}>{t`Decide later`}</Button>
        {nexusKey === null || keep !== nexusKey ? (
          <DisabledReason title={lockedTitle} disabled={locked}>
            <Button
              variant={nexusKey === null ? 'contained' : 'outlined'}
              disabled={locked}
              onClick={() => {
                keepCopy(dup, keep).catch(reportUnexpected)
              }}
            >
              {t`Keep this one`}
            </Button>
          </DisabledReason>
        ) : null}
        {nexusKey === null ? null : (
          <DisabledReason title={lockedTitle} disabled={locked}>
            <Button
              variant="contained"
              disabled={locked}
              onClick={() => {
                keepCopy(dup, nexusKey).catch(reportUnexpected)
              }}
            >
              {t`Keep the Nexus copy`}
            </Button>
          </DisabledReason>
        )}
      </DialogActions>
    </>
  )
}

export function DuplicateDialog({ profileName }: { profileName: string }) {
  const dup = useMods((s) => s.resolving)
  const resolve = useMods((s) => s.resolve)
  return (
    <Dialog
      open={dup !== null}
      onClose={() => resolve(null)}
      maxWidth={false}
      slotProps={{ paper: { sx: { width: 780, maxWidth: 'calc(100% - 32px)' } } }}
    >
      {dup ? <Resolver key={dup.uniqueId} dup={dup} profileName={profileName} /> : null}
    </Dialog>
  )
}
