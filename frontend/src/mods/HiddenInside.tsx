import { useLingui } from '@lingui/react/macro'
import { Box, Button, List, ListItem, ListItemText, Typography } from '@mui/material'
import { useState } from 'react'
import type {
  HiddenMod,
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  DotHiddenMods,
  ShowHiddenMod,
  UnhideMod,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { useLoaded } from '../shell/useLoaded.ts'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { heading } from './paper.ts'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

// Mods SMAPI skips because a folder on their path starts with a dot, listed under the entry that holds them.
export function HiddenInside({ mod, profile }: { mod: Mod; profile: Profile }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const on = useSettings((s) => s.games?.[game]?.showDotHiddenMods === true)
  const [unhiding, setUnhiding] = useState<HiddenMod | null>(null)
  const [busy, run] = usePending()
  const locked = useLocked()
  const { data: hidden } = useLoaded<HiddenMod[]>(
    !on || game === '' || profile.updated === ''
      ? null
      : () => DotHiddenMods(game, profile.id, mod.key).then((list) => list ?? []),
    [on, game, profile.id, profile.updated, mod.key],
    [],
    reportUnexpected,
  )
  if (hidden.length === 0) {
    return null
  }
  return (
    <Box>
      <Typography component="h3" sx={heading}>
        {t`Hidden inside this mod`}
      </Typography>
      <List dense={true} disablePadding={true}>
        {hidden.map((h) => (
          <ListItem key={`${h.folder}-${h.id}`} disableGutters={true} sx={{ py: 0.25 }}>
            <ListItemText
              primary={h.version === '' ? h.name : `${h.name} · ${h.version}`}
              secondary={h.folder}
              slotProps={{
                primary: { sx: { fontSize: 13, overflowWrap: 'anywhere' } },
                secondary: { sx: { fontSize: 12, overflowWrap: 'anywhere' } },
              }}
            />
            <Button
              size="small"
              variant="text"
              color="inherit"
              onClick={() => {
                ShowHiddenMod(game, profile.id, mod.key, h.folder).catch(
                  reportError(t`Could not open the folder`),
                )
              }}
            >
              {t`Open folder`}
            </Button>
            <DisabledReason title={t`Stop the game first.`} disabled={locked}>
              <Button
                size="small"
                variant="text"
                color="inherit"
                disabled={locked}
                onClick={() => setUnhiding(h)}
              >
                {t`Unhide`}
              </Button>
            </DisabledReason>
          </ListItem>
        ))}
      </List>
      <Typography sx={{ fontSize: 12, color: 'text.secondary', mt: 0.5 }}>
        {t`SMAPI skips folders whose names start with a dot.`}
      </Typography>
      <ConfirmDialog
        open={unhiding !== null}
        title={t`Unhide ${unhiding?.name ?? ''}?`}
        body={t`Mortar renames the folder ${unhiding?.folder ?? ''} without its leading dots, so SMAPI loads the mod.`}
        confirmLabel={t`Unhide`}
        busy={busy}
        onCancel={() => setUnhiding(null)}
        onConfirm={() => {
          const target = unhiding
          if (target === null) {
            return
          }
          run(
            async () => {
              useProfiles
                .getState()
                .replace(await UnhideMod(game, profile.id, mod.key, target.folder))
              await useMods.getState().load()
              setUnhiding(null)
            },
            { errorTitle: t`Could not unhide the mod` },
          )
        }}
      />
    </Box>
  )
}
