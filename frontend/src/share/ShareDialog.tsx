import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Dialog, IconButton, Typography } from '@mui/material'
import { Clipboard } from '@wailsio/runtime'
import { Copy, MessageSquare, Save, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Saved } from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/models.ts'
import {
  SaveFile,
  Share,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/service.ts'
import { heading, paper } from '../mods/paper.ts'
import { useProfiles } from '../profiles/store.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { type MeterLevel, meter, type ShownInfo, shownInfo, suggestFile } from './logic.ts'
import { SharedMods } from './SharedMods.tsx'
import { useShareDialog } from './store.ts'
import { TabPills } from './TabPills.tsx'

type Tab = 'link' | 'file'

const PERCENT = 100
const PAGE_HOST = 'mortar.rethunk.tech'

const LEVEL_COLOUR: Record<MeterLevel, string> = {
  ok: 'success.main',
  warn: 'warning.main',
  over: 'error.main',
}

function Meter({ info }: { info: ShownInfo }) {
  const { t } = useLingui()
  const { ratio, level } = meter(info.length, info.limit)
  const length = info.length.toLocaleString()
  const limit = info.limit.toLocaleString()
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.75 }}>
      <Box
        role="meter"
        aria-label={t`Link length`}
        aria-valuemin={0}
        aria-valuemax={info.limit}
        aria-valuenow={info.length}
        sx={{
          height: 8,
          borderRadius: '4px',
          bgcolor: 'rgba(255,255,255,0.12)',
          overflow: 'hidden',
        }}
      >
        <Box sx={{ width: `${ratio * PERCENT}%`, height: '100%', bgcolor: LEVEL_COLOUR[level] }} />
      </Box>
      <Typography sx={{ fontSize: 13, color: level === 'over' ? 'error.light' : 'text.secondary' }}>
        {t`${length} of ${limit} characters, the most a Discord message holds`}
      </Typography>
    </Box>
  )
}

function PagePreview({ info }: { info: ShownInfo }) {
  const { t } = useLingui()
  const art = useProfiles((s) => s.game?.artUrl)
  const gameName = useProfiles((s) => s.game?.name ?? '')
  const mods = plural(info.count, { one: '# mod', other: '# mods' })
  return (
    <Box
      component="aside"
      aria-label={t`What the person opening the link sees`}
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: 1.5,
        p: 2.5,
        bgcolor: 'rgba(0,0,0,0.25)',
        borderLeft: '1px solid rgba(255,255,255,0.1)',
      }}
    >
      <Typography sx={heading}>{t`What they see`}</Typography>
      <Box sx={{ bgcolor: 'rgba(55,55,65,0.9)', borderRadius: '8px', overflow: 'hidden' }}>
        <Typography sx={{ px: 1.5, py: 0.75, fontSize: 12, color: 'text.secondary' }}>
          {PAGE_HOST}
        </Typography>
        {art ? (
          <Box
            component="img"
            src={art}
            alt=""
            sx={{ width: '100%', height: 110, objectFit: 'cover', display: 'block' }}
          />
        ) : null}
        <Box sx={{ p: 2, display: 'flex', flexDirection: 'column', gap: 1 }}>
          <Typography sx={{ fontSize: 18, fontWeight: 700 }}>{info.name}</Typography>
          <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
            {t`${gameName} · ${mods}`}
          </Typography>
          <Box sx={{ display: 'flex', gap: 1, mt: 0.5 }}>
            <Box
              sx={{
                px: 1.5,
                py: 0.75,
                borderRadius: '6px',
                bgcolor: 'primary.main',
                color: 'primary.contrastText',
                fontSize: 13,
                fontWeight: 600,
                whiteSpace: 'nowrap',
              }}
            >
              {t`Open in Mortar`}
            </Box>
            <Box
              sx={{
                px: 1.5,
                py: 0.75,
                borderRadius: '6px',
                bgcolor: 'rgba(255,255,255,0.12)',
                fontSize: 13,
                whiteSpace: 'nowrap',
              }}
            >
              {t`Get Mortar`}
            </Box>
          </Box>
        </Box>
      </Box>
      <Typography sx={{ fontSize: 13, color: 'text.secondary', lineHeight: 1.5 }}>
        {t`The page reads the profile from the link itself. Nothing is uploaded or stored on a server.`}
      </Typography>
    </Box>
  )
}

function LinkTab({ info, onFile }: { info: ShownInfo; onFile: () => void }) {
  const { t } = useLingui()
  const gameName = useProfiles((s) => s.game?.name ?? '')
  const copy = (text: string, done: string) => {
    Clipboard.SetText(text).then(
      () => useToasts.getState().push({ kind: 'success', title: done }),
      reportUnexpected,
    )
  }
  const mods = plural(info.count, { one: '# mod', other: '# mods' })
  const message = t`Try my Mortar profile "${info.name}" for ${gameName} (${mods}): ${info.web}`
  const large = suggestFile(info.count, info.tooLarge)
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      {info.tooLarge ? (
        <Typography sx={{ fontSize: 14, color: 'warning.light' }}>
          {t`This profile is too large for a link.`}
        </Typography>
      ) : (
        <>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Box
              sx={{
                flex: 1,
                minWidth: 0,
                display: 'flex',
                alignItems: 'center',
                height: 46,
                px: 1.5,
                bgcolor: 'rgba(0,0,0,0.45)',
                border: '1px solid rgba(255,255,255,0.15)',
                borderRadius: '6px',
                fontFamily: '"IBM Plex Mono", monospace',
                fontSize: 13,
                userSelect: 'text',
              }}
            >
              <Typography noWrap={true} sx={{ font: 'inherit' }}>
                {info.web}
              </Typography>
            </Box>
            <Button
              variant="contained"
              startIcon={<Copy size={16} />}
              onClick={() => copy(info.web, t`Link copied`)}
              sx={{ height: 46 }}
            >
              {t`Copy link`}
            </Button>
          </Box>
          <Meter info={info} />
          <Box>
            <Button
              variant="outlined"
              color="inherit"
              startIcon={<MessageSquare size={16} />}
              onClick={() => copy(message, t`Message copied`)}
            >
              {t`Copy as a message`}
            </Button>
          </Box>
        </>
      )}
      {large ? (
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 1.5,
            p: 1.5,
            bgcolor: 'rgba(243,180,22,0.14)',
            border: '1px solid rgba(243,180,22,0.5)',
            borderRadius: '6px',
          }}
        >
          <Typography sx={{ flex: 1, fontSize: 14 }}>
            {t`A .mortar file suits a profile this size better, and carries your mod settings too.`}
          </Typography>
          <Button variant="outlined" color="inherit" onClick={onFile}>
            {t`Use a file`}
          </Button>
        </Box>
      ) : null}
      <SharedMods info={info} />
    </Box>
  )
}

function FileTab({ info, game, profileId }: { info: ShownInfo; game: string; profileId: string }) {
  const { t } = useLingui()
  const [saved, setSaved] = useState<Saved | null>(null)
  const [busy, setBusy] = useState(false)
  const save = async () => {
    setBusy(true)
    try {
      const result = await SaveFile(game, profileId)
      if (result.path) {
        setSaved(result)
        useToasts.getState().push({ kind: 'success', title: t`File saved` })
      }
    } catch (e) {
      useToasts
        .getState()
        .push({ kind: 'error', title: t`Could not save the file`, body: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const skipped = saved?.skipped ?? []
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      <Typography sx={{ fontSize: 14, lineHeight: 1.5 }}>
        {t`A .mortar file holds the same mods as the link, plus the settings files (.json) each mod has written. Send it to someone, and they open it from Mortar's Import.`}
      </Typography>
      <Box>
        <Button
          variant="contained"
          startIcon={<Save size={16} />}
          disabled={busy}
          onClick={() => {
            save().catch(reportUnexpected)
          }}
        >
          {t`Save…`}
        </Button>
      </Box>
      {saved?.path ? (
        <Typography sx={{ fontSize: 13, color: 'text.secondary', wordBreak: 'break-all' }}>
          {t`Saved to ${saved.path}`}
        </Typography>
      ) : null}
      {skipped.length > 0 ? (
        <Box>
          <Typography sx={{ fontSize: 13, fontWeight: 600 }}>
            {plural(skipped.length, {
              one: '# settings file was left out (too large or an unusual name):',
              other: '# settings files were left out (too large or an unusual name):',
            })}
          </Typography>
          <Box component="ul" sx={{ m: 0, pl: 2.5, fontSize: 13, color: 'text.secondary' }}>
            {skipped.map((p) => (
              <li key={p}>{p}</li>
            ))}
          </Box>
        </Box>
      ) : null}
      <SharedMods info={info} />
    </Box>
  )
}

export function ShareDialog() {
  const { t } = useLingui()
  const profileId = useShareDialog((s) => s.profileId)
  const close = useShareDialog((s) => s.close)
  const game = useProfiles((s) => s.game?.id ?? 'stardew')
  const art = useProfiles((s) => s.game?.artUrl)
  const [tab, setTab] = useState<Tab>('link')
  const [info, setInfo] = useState<ShownInfo | null>(null)
  useEffect(() => {
    if (!profileId) {
      return
    }
    let stale = false
    setInfo(null)
    setTab('link')
    Share(game, profileId).then(
      (next) => {
        if (!stale) {
          setInfo(shownInfo(next))
          setTab(suggestFile(next.count, next.tooLarge) ? 'file' : 'link')
        }
      },
      (e: unknown) => {
        useToasts
          .getState()
          .push({ kind: 'error', title: t`Could not build the link`, body: errorMessage(e) })
        close()
      },
    )
    return () => {
      stale = true
    }
  }, [profileId, game, close, t])
  return (
    <Dialog
      open={profileId !== '' && info !== null}
      onClose={close}
      maxWidth={false}
      slotProps={{
        paper: {
          ...paper,
          sx: {
            ...paper.sx,
            width: 'min(980px, calc(100% - 48px))',
            height: 'min(620px, calc(100% - 48px))',
            overflow: 'hidden',
            borderRadius: '10px',
          },
        },
      }}
    >
      {info ? (
        <Box
          role="dialog"
          aria-label={t`Share ${info.name}`}
          sx={{
            display: 'grid',
            gridTemplateColumns: tab === 'link' ? 'minmax(0, 1fr) 340px' : 'minmax(0, 1fr)',
            height: '100%',
            minHeight: 0,
          }}
        >
          <Box sx={{ display: 'flex', flexDirection: 'column', minWidth: 0, minHeight: 0 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.75, p: '20px 24px 0' }}>
              {art ? (
                <Box
                  component="img"
                  src={art}
                  alt=""
                  sx={{ width: 64, height: 64, objectFit: 'cover', borderRadius: '8px' }}
                />
              ) : null}
              <Box sx={{ flex: 1, minWidth: 0 }}>
                <Typography
                  sx={{ fontSize: 13, color: 'text.secondary' }}
                >{t`Share profile`}</Typography>
                <Typography noWrap={true} sx={{ fontSize: 24, fontWeight: 700 }}>
                  {info.name}
                </Typography>
              </Box>
              <IconButton aria-label={t`Close`} onClick={close} sx={{ alignSelf: 'flex-start' }}>
                <X size={18} />
              </IconButton>
            </Box>
            <Box sx={{ m: '18px 24px 0', display: 'flex' }}>
              <TabPills
                value={tab}
                onChange={setTab}
                label={t`How to share`}
                options={[
                  { value: 'link', label: t`Link` },
                  { value: 'file', label: t`.mortar file with settings` },
                ]}
              />
            </Box>
            <Box sx={{ p: '18px 24px', overflowY: 'auto', flex: 1, minHeight: 0 }}>
              {tab === 'link' ? <LinkTab info={info} onFile={() => setTab('file')} /> : null}
              {tab === 'file' ? <FileTab info={info} game={game} profileId={profileId} /> : null}
            </Box>
          </Box>
          {tab === 'link' ? <PagePreview info={info} /> : null}
        </Box>
      ) : null}
    </Dialog>
  )
}
