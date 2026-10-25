import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, FormControlLabel, Menu, Typography } from '@mui/material'
import { Download, UserRound } from 'lucide-react'
import { useState } from 'react'
import type { Arrival } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nxmsvc/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { NewProfileDialog } from '../game/NewProfileDialog.tsx'
import { formatKb } from '../i18n/bytes.ts'
import { nexusEntryOf, useNexusDetails } from '../mods/nexusDetails.ts'
import { listProfiles } from '../profiles/listProfiles.ts'
import { useProfiles } from '../profiles/store.ts'
import { gamePrefs } from '../settings/gamePrefs.ts'
import { useSettings } from '../settings/store.ts'
import { AppDialog, DialogFooter, ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { ErrorRetry } from '../shell/ErrorRetry.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { useLoaded } from '../shell/useLoaded.ts'
import { type InlineError, inlineError, reportUnexpected } from '../toasts/report.ts'
import {
  arrivalGame,
  arrivalName,
  arrivalSite,
  fallbackName,
  packageName,
  useNxm,
} from './store.ts'

function useModName(arrival: Arrival): string {
  const { data: name } = useLoaded(
    () => arrivalName(arrival),
    [arrival],
    packageName(arrival) || fallbackName(arrival.link.modId),
    reportUnexpected,
  )
  return name
}

// The linked file as the mod's Nexus details list it, once they are read.
const useArrivalFile = (game: string, arrival: Arrival) =>
  useNexusDetails((s) =>
    nexusEntryOf(s.byKey, game, arrival.link.modId)?.details?.files?.find(
      (f) => f.fileId === arrival.link.fileId,
    ),
  )

function ArrivalPrompt({ arrival }: { arrival: Arrival }) {
  const { t } = useLingui()
  const openGameId = useProfiles((s) => s.game?.id)
  const game = arrivalGame(arrival, openGameId)
  const site = arrivalSite(arrival)
  const choose = useNxm((s) => s.choose)
  const arrivals = useNxm((s) => s.arrivals)
  const dismiss = useNxm((s) => s.dismiss)
  const lastId = useSettings((s) => gamePrefs(s, game).nxmDefaultProfile || s.lastProfile?.[game])
  const name = useModName(arrival)
  const file = useArrivalFile(game, arrival)
  // New profile creates in the game the profiles store has open, so it is offered only when that is this game.
  const canCreate = useProfiles((s) => s.game?.id === game)
  const gameName = useProfiles((s) => (s.game?.id === game ? s.game.name : game))
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [creating, setCreating] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<InlineError | null>(null)
  const [applyAll, setApplyAll] = useState(false)
  const listed = useLoaded<Profile[] | null>(
    () => listProfiles(game).then((list) => list ?? []),
    [game],
    null,
    reportUnexpected,
  )
  const load = listed.reload
  // A list that cannot be read offers New profile rather than waiting on a spinner.
  const profiles = listed.error === null ? listed.data : (listed.data ?? [])
  const open = profiles?.find((p) => p.id === lastId) ?? profiles?.[0]
  const others = profiles?.filter((p) => p.id !== open?.id) ?? []
  const pick = (profile: string) => {
    if (busy) {
      return
    }
    setBusy(true)
    setError(null)
    const ids = applyAll ? arrivals.map((a) => a.id) : [arrival.id]
    Promise.all(ids.map((id) => choose(id, profile)))
      .catch((e: unknown) => setError(inlineError(e)))
      .finally(() => setBusy(false))
  }
  let text = t`You started this download on ${{ source: site }}. Choose the profile it goes into.`
  if (profiles === null) {
    text = t`Loading profiles…`
  } else if (profiles.length === 0) {
    text = canCreate
      ? t`You started this download on ${{ source: site }}. Create a profile to put it in first.`
      : t`You started this download on ${{ source: site }}, but you have no ${{ game: gameName }} profile yet.`
  }
  return (
    <>
      <AppDialog
        open={true}
        onClose={() => (busy ? undefined : dismiss(arrival.id))}
        closable={false}
        size="sm"
        title={t`Install ${name}?`}
        subtitle={file ? `${file.fileName} · ${formatKb(file.sizeKb)}` : undefined}
        icon={
          <ToneIcon tone="neutral">
            <Download size={22} />
          </ToneIcon>
        }
        footer={
          <DialogFooter>
            <Button disabled={busy} onClick={() => dismiss(arrival.id)}>
              {t`Ignore`}
            </Button>
            {others.length > 0 ? (
              <Button
                disabled={busy}
                aria-haspopup="menu"
                aria-expanded={anchor !== null}
                onClick={(e) => setAnchor(e.currentTarget)}
              >
                {t`Other profile…`}
              </Button>
            ) : null}
            {open ? (
              <Button variant="contained" disabled={busy} onClick={() => pick(open.id)}>
                {t`Install into ${{ name: open.name }}`}
              </Button>
            ) : null}
            {profiles?.length === 0 && canCreate ? (
              <Button variant={open ? 'text' : 'contained'} onClick={() => setCreating(true)}>
                {t`New profile`}
              </Button>
            ) : null}
          </DialogFooter>
        }
      >
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
          {arrivals.length > 1 ? (
            <Typography sx={{ color: 'text.secondary' }}>
              {t`${{ count: arrivals.length - 1 }} more waiting`}
            </Typography>
          ) : null}
          {profiles === null ? (
            <LoadingRow>{text}</LoadingRow>
          ) : (
            <Typography sx={{ color: 'text.secondary' }}>{text}</Typography>
          )}
          {error ? (
            <ErrorRetry
              error={error}
              onRetry={() => {
                if (open) {
                  pick(open.id)
                }
              }}
            />
          ) : null}
          {arrivals.length > 1 ? (
            <FormControlLabel
              control={
                <Checkbox checked={applyAll} onChange={(_, checked) => setApplyAll(checked)} />
              }
              label={t`Apply this profile to all waiting`}
            />
          ) : null}
        </Box>
      </AppDialog>
      <Menu anchorEl={anchor} open={anchor !== null} onClose={() => setAnchor(null)}>
        {others.map((p) => (
          <MenuAction
            key={p.id}
            icon={<UserRound size={16} aria-hidden={true} />}
            label={p.name}
            onClick={() => {
              setAnchor(null)
              pick(p.id)
            }}
          />
        ))}
      </Menu>
      <NewProfileDialog
        open={creating}
        onClose={() => {
          setCreating(false)
          load()
        }}
      />
    </>
  )
}

// One prompt at a time; the next link's prompt follows once this one is answered.
export function ArrivalDialog() {
  const arrival = useNxm((s) => s.arrivals[0])
  return arrival ? <ArrivalPrompt key={arrival.id} arrival={arrival} /> : null
}
