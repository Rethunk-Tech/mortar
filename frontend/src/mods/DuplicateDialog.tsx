import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Radio, RadioGroup, Typography } from '@mui/material'
import { Copy as CopyIcon } from 'lucide-react'
import { useState } from 'react'
import type {
  Copy,
  Duplicate,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { SMAPI } from '../brand/productNames.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { useGameInfo } from '../games/info.ts'
import { listNames } from '../i18n/list.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { AppDialog, DialogFooter, ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { LetterTile } from './LetterTile.tsx'
import { nexusKeepKey, preselect } from './lookup.ts'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

function CopyOption({ copy, dup, differ }: { copy: Copy; dup: Duplicate; differ: boolean }) {
  const { t } = useLingui()
  const source = (() => {
    if (copy.nexus) {
      return t`From ${{ source: sourceLabel('nexus') }}`
    }
    if (copy.source === 'smapi') {
      return t`Bundled with ${{ loader: SMAPI }}`
    }
    return copy.source === 'local'
      ? t`From an archive`
      : t`From ${{ source: sourceLabel(copy.source) }}`
  })()
  const checked = copy.source !== 'local' && copy.source !== 'smapi'
  const needed = copy.needed ?? []
  const tooOld = copy.tooOld ?? []
  const lines = [
    differ && copy.newest ? t`Newer.` : '',
    differ && !copy.newest ? t`Older.` : '',
    checked ? t`Gets update checks.` : t`No update checks, and it cannot go in a share link.`,
    needed.length > 0 ? t`${{ names: listNames(needed) }} needs this version or newer.` : '',
    tooOld.length > 0 ? t`${{ names: listNames(tooOld) }} needs a newer version than this.` : '',
  ].filter(Boolean)
  return (
    <Box
      component="label"
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: space.gap,
        p: space.pad,
        cursor: 'pointer',
        bgcolor: 'var(--mortar-raised)',
        border: '2px solid transparent',
        borderRadius: '8px',
        '&:has(input:checked)': { borderColor: 'primary.main' },
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
        <Radio
          value={copy.key}
          sx={{ p: 0 }}
          slotProps={{
            input: { 'aria-label': t`${source}, version ${{ version: copy.version }}` },
          }}
        />
        <LetterTile mod={{ id: dup.id, name: copy.name }} size={40} />
        <Box sx={{ minWidth: 0 }}>
          <Typography sx={{ fontSize: 'lead.fontSize', fontWeight: 700 }}>{source}</Typography>
          <Typography sx={{ fontSize: 'base.fontSize', color: 'text.secondary' }}>
            {t`Version ${{ version: copy.version }}`}
          </Typography>
        </Box>
      </Box>
      <Typography sx={{ fontSize: 'base.fontSize', lineHeight: 1.5 }}>{lines.join(' ')}</Typography>
    </Box>
  )
}

function Resolver({
  dup,
  profileName,
  onClose,
}: {
  dup: Duplicate
  profileName: string
  onClose: () => void
}) {
  const { t } = useLingui()
  const loader = useGameInfo()?.loader ?? ''
  const keepCopy = useMods((s) => s.keepCopy)
  const locked = useLocked()
  const lockedTitle = t`Stop the game to change mods.`
  const copies = dup.copies ?? []
  const [keep, setKeep] = useState(preselect(copies))
  const nexusKey = nexusKeepKey(copies)
  const differ = copies.some((c) => !c.newest)
  return (
    <AppDialog
      open={true}
      onClose={onClose}
      size="lg"
      title={plural(
        { count: copies.length },
        { one: `# copy of ${{ name: dup.name }}`, other: `# copies of ${{ name: dup.name }}` },
      )}
      subtitle={t`${{ profile: profileName }} has ${{ name: dup.name }} more than once. ${loader} loads only one, so pick which to keep. The others are disabled, not removed.`}
      icon={
        <ToneIcon tone="warning">
          <CopyIcon size={22} />
        </ToneIcon>
      }
      footer={
        <DialogFooter>
          <Button onClick={onClose}>{t`Decide later`}</Button>
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
                {t`Keep the ${{ source: sourceLabel('nexus') }} copy`}
              </Button>
            </DisabledReason>
          )}
        </DialogFooter>
      }
    >
      <RadioGroup
        aria-label={t`Copy to keep`}
        value={keep}
        onChange={(e) => setKeep(e.target.value)}
        sx={{
          display: 'grid',
          gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
          gap: space.gap,
        }}
      >
        {copies.map((c) => (
          <CopyOption key={c.key} copy={c} dup={dup} differ={differ} />
        ))}
      </RadioGroup>
    </AppDialog>
  )
}

export function DuplicateDialog({ profileName }: { profileName: string }) {
  const dup = useMods((s) => s.resolving)
  const resolve = useMods((s) => s.resolve)
  return dup ? (
    <Resolver key={dup.id} dup={dup} profileName={profileName} onClose={() => resolve(null)} />
  ) : null
}
