import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
  Typography,
} from '@mui/material'
import { Link2 } from 'lucide-react'
import { memo, useState } from 'react'
import { create } from 'zustand'
import { SetNexusPage } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { type InlineError, inlineError, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useLocked } from './useLocked.ts'

interface Target {
  game: string
  profileId: string
  modKey: string
  name: string
}

const useNexusPage = create<{ target: Target | null; show: (t: Target | null) => void }>((set) => ({
  target: null,
  show: (target) => set({ target }),
}))

// The mod menu's item for a mod with no source, or one on Nexus; a mod from another site keeps its source.
export function NexusPageMenuItem({ mod, close }: { mod: Mod; close: () => void }) {
  const { t } = useLingui()
  const locked = useLocked()
  const game = useProfiles((s) => s.game)
  const profile = useProfiles(openProfileOf)
  const kind = (profile?.entries ?? []).find((e) => e.key === mod.key)?.source.kind
  if (!(game?.sources?.includes('nexus') && profile) || (kind !== 'local' && kind !== 'nexus')) {
    return null
  }
  return (
    <MenuAction
      disabled={locked}
      icon={<Link2 size={16} />}
      label={t`Set ${{ source: sourceLabel('nexus') }} page…`}
      onClick={() => {
        close()
        useNexusPage
          .getState()
          .show({ game: game.id, profileId: profile.id, modKey: mod.key, name: mod.name })
      }}
    />
  )
}

// Names the Nexus page one mod comes from, for a mod Mortar could not place or placed on the wrong page.
// Takes no props, so a re-render of the tab around it skips it.
export const SetNexusPageDialog = memo(function SetNexusPageDialogView() {
  const target = useNexusPage((s) => s.target)
  const onClose = () => useNexusPage.getState().show(null)
  const { game = '', profileId = '', modKey = '', name = '' } = target ?? {}
  const open = target !== null
  const { t } = useLingui()
  const [link, setLink] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<InlineError | null>(null)
  const close = () => {
    setLink('')
    setError(null)
    onClose()
  }
  const save = async () => {
    setBusy(true)
    setError(null)
    try {
      const profile = await SetNexusPage(game, profileId, modKey, link.trim())
      useProfiles.getState().replace(profile)
      useToasts.getState().push({
        kind: 'success',
        title: t`${name} now comes from that ${{ source: sourceLabel('nexus') }} page`,
        changes: [profile.lastChange ?? ''],
      })
      close()
    } catch (e) {
      setError(inlineError(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog open={open} onClose={close} fullWidth={true} maxWidth="sm">
      <DialogTitle>{t`Set the ${{ source: sourceLabel('nexus') }} page of ${name}`}</DialogTitle>
      <DialogContent>
        <Typography sx={{ mb: 1.5, fontSize: 14, color: 'text.secondary' }}>
          {t`Paste the address of the mod's page on ${{ source: sourceLabel('nexus') }}. To name the exact file, open that file's Manual Download on the page's Files tab and paste that address instead.`}
        </Typography>
        <TextField
          autoFocus={true}
          fullWidth={true}
          size="small"
          label={t`${{ source: sourceLabel('nexus') }} page address`}
          placeholder="https://www.nexusmods.com/…/mods/…"
          value={link}
          disabled={busy}
          error={error !== null}
          helperText={error?.message}
          onChange={(e) => setLink(e.target.value)}
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={close} disabled={busy}>
          {t`Cancel`}
        </Button>
        <Button
          variant="contained"
          disabled={busy || link.trim() === ''}
          onClick={() => save().catch(reportUnexpected)}
        >
          {t`Set page`}
        </Button>
      </DialogActions>
    </Dialog>
  )
})
