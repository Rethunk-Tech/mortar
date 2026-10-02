import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  Divider,
  IconButton,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Tooltip,
  Typography,
} from '@mui/material'
import { Clipboard } from '@wailsio/runtime'
import { Check, Copy, FileText, List, MessageSquare, Save, Type, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Saved } from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/models.ts'
import {
  SaveFile,
  Share,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/service.ts'
import { Logo } from '../brand/Logo.tsx'
import { heading, paper } from '../mods/paper.ts'
import { useProfiles } from '../profiles/store.ts'
import { TipBanner } from '../tips/TipBanner.tsx'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { type MeterLevel, meter, type ShownInfo, shownInfo, suggestFile } from './logic.ts'
import { formatModList, listItems, type ModListFormat } from './modList.ts'
import { SharedMods } from './SharedMods.tsx'
import { useShareDialog } from './store.ts'
import { TabPills } from './TabPills.tsx'

type Tab = 'link' | 'file'

const PERCENT = 100
const PAGE_HOST = 'mortar.rethunk.tech'

const LEVEL_TEXT: Record<MeterLevel, string> = {
  ok: '#6ff5a8',
  warn: 'warning.main',
  over: 'error.light',
}

const LEVEL_COLOUR: Record<MeterLevel, string> = {
  ok: 'success.main',
  warn: 'warning.main',
  over: 'error.main',
}

function Meter({ info }: { info: ShownInfo }) {
  const { t, i18n } = useLingui()
  const { ratio, level } = meter(info.length, info.limit)
  const length = info.length.toLocaleString(i18n.locale)
  const limit = info.limit.toLocaleString(i18n.locale)
  const verdict = {
    ok: t`${length} of ${limit} characters · fits one Discord message`,
    warn: t`${length} of ${limit} characters · close to the Discord message limit`,
    over: t`${length} of ${limit} characters · too long for one Discord message`,
  }[level]
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25 }}>
      <Box
        role="meter"
        aria-label={t`Link length`}
        aria-valuemin={0}
        aria-valuemax={info.limit}
        aria-valuenow={info.length}
        sx={{
          flex: 1,
          height: 8,
          borderRadius: '4px',
          bgcolor: 'rgba(255,255,255,0.1)',
          overflow: 'hidden',
        }}
      >
        <Box sx={{ width: `${ratio * PERCENT}%`, height: '100%', bgcolor: LEVEL_COLOUR[level] }} />
      </Box>
      <Typography sx={{ fontSize: 13, whiteSpace: 'nowrap', color: LEVEL_TEXT[level] }}>
        {verdict}
      </Typography>
    </Box>
  )
}

function PagePreview({ info, onClose }: { info: ShownInfo; onClose: () => void }) {
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
        bgcolor: 'rgba(24,24,30,0.9)',
        borderLeft: '1px solid rgba(255,255,255,0.08)',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center' }}>
        <Typography sx={{ ...heading, flex: 1 }}>{t`What they see`}</Typography>
        <Tooltip title={t`Close`}>
          <IconButton aria-label={t`Close`} onClick={onClose} sx={{ width: 40, height: 40 }}>
            <X size={16} />
          </IconButton>
        </Tooltip>
      </Box>
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: 1.5,
          p: '18px',
          bgcolor: '#f4f1ea',
          color: '#1b1a17',
          borderRadius: '8px',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, fontSize: 13, fontWeight: 700 }}>
          <Logo size={16} fill="#1b1a17" />
          {PAGE_HOST}
        </Box>
        {art ? (
          <Box
            component="img"
            src={art}
            alt=""
            sx={{ width: '100%', height: 96, objectFit: 'cover', borderRadius: '6px' }}
          />
        ) : null}
        <Typography
          noWrap={true}
          title={info.name}
          sx={{ fontSize: 20, fontWeight: 700, color: 'inherit' }}
        >
          {info.name}
        </Typography>
        <Typography sx={{ fontSize: 13, color: 'inherit' }}>{t`${gameName} · ${mods}`}</Typography>
        <Box
          sx={{
            display: 'grid',
            placeItems: 'center',
            height: 40,
            bgcolor: '#1b1a17',
            color: '#f4f1ea',
            borderRadius: '6px',
            fontSize: 14,
            fontWeight: 700,
            whiteSpace: 'nowrap',
          }}
        >
          {t`Open in Mortar`}
        </Box>
        <Box
          sx={{
            display: 'grid',
            placeItems: 'center',
            height: 40,
            border: '1px solid #1b1a17',
            borderRadius: '6px',
            fontSize: 14,
            whiteSpace: 'nowrap',
          }}
        >
          {t`Get Mortar`}
        </Box>
      </Box>
      <Typography sx={{ fontSize: 12, color: 'text.secondary', lineHeight: 1.5 }}>
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
    <>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.25, p: '18px 24px' }}>
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
                sx={{ height: 46, px: '18px', fontSize: 14 }}
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
                sx={{ height: 40, px: '14px', fontSize: 14 }}
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
      </Box>
      <Divider sx={{ mx: 3 }} />
      <SharedMods info={info} notIn={t`Not in the link`} />
    </>
  )
}

function copyText(text: string, done: string) {
  Clipboard.SetText(text).then(
    () => useToasts.getState().push({ kind: 'success', title: done }),
    reportUnexpected,
  )
}

function CopyModList() {
  const { t } = useLingui()
  const profileId = useShareDialog((s) => s.profileId)
  const keys = useShareDialog((s) => s.keys)
  const format = useShareDialog((s) => s.listFormat)
  const setFormat = useShareDialog((s) => s.setListFormat)
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === profileId))
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const items = listItems(profile, keys)
  const labels = { enabled: t`Enabled`, disabled: t`Disabled` }
  const parts = formatModList(format, items, labels)
  const options: { id: ModListFormat; label: string; icon: typeof FileText }[] = [
    { id: 'markdown', label: t`Markdown`, icon: FileText },
    { id: 'plain', label: t`Plain text`, icon: Type },
    { id: 'discord', label: t`Discord`, icon: MessageSquare },
  ]
  return (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 1 }}>
      {format === 'discord' && parts.length > 1 ? (
        parts.map((part) => (
          <Button
            key={part.id}
            variant="outlined"
            color="inherit"
            startIcon={<Copy size={16} />}
            onClick={() => copyText(part.text, t`Copied part ${part.n}`)}
            sx={{ height: 40, px: '14px', fontSize: 14, whiteSpace: 'nowrap' }}
          >
            {t`Copy part ${part.n}`}
          </Button>
        ))
      ) : (
        <Button
          variant="outlined"
          color="inherit"
          startIcon={<List size={16} />}
          onClick={() => copyText(parts[0]?.text ?? '', t`Mod list copied`)}
          sx={{ height: 40, px: '14px', fontSize: 14, whiteSpace: 'nowrap' }}
        >
          {t`Copy mod list`}
        </Button>
      )}
      <Button
        variant="outlined"
        color="inherit"
        aria-label={t`Mod list format`}
        onClick={(e) => setAnchor(e.currentTarget)}
        sx={{ height: 40, px: '14px', fontSize: 14, whiteSpace: 'nowrap' }}
      >
        {options.find((o) => o.id === format)?.label}
      </Button>
      <Menu
        open={Boolean(anchor)}
        anchorEl={anchor}
        onClose={() => setAnchor(null)}
        transitionDuration={0}
        slotProps={{ paper }}
      >
        {options.map((o) => {
          const Icon = o.icon
          return (
            <MenuItem
              key={o.id}
              selected={o.id === format}
              onClick={() => {
                setFormat(o.id)
                setAnchor(null)
              }}
            >
              <ListItemIcon sx={{ color: 'inherit' }}>
                {o.id === format ? <Check size={16} /> : <Icon size={16} />}
              </ListItemIcon>
              <ListItemText>{o.label}</ListItemText>
            </MenuItem>
          )
        })}
      </Menu>
    </Box>
  )
}

function FileTab({
  info,
  game,
  profileId,
  keys,
}: {
  info: ShownInfo
  game: string
  profileId: string
  keys: string[]
}) {
  const { t } = useLingui()
  const [saved, setSaved] = useState<Saved | null>(null)
  const [busy, setBusy] = useState(false)
  const save = async () => {
    setBusy(true)
    try {
      const result = await SaveFile(game, profileId, keys)
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
    <>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.25, p: '18px 24px' }}>
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
      </Box>
      <Divider sx={{ mx: 3 }} />
      <SharedMods info={info} notIn={t`Not in the file`} />
    </>
  )
}

export function ShareDialog() {
  const { t } = useLingui()
  const profileId = useShareDialog((s) => s.profileId)
  const keys = useShareDialog((s) => s.keys)
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
    Share(game, profileId, keys).then(
      (next) => {
        if (!stale) {
          setInfo(shownInfo(next))
          setTab(suggestFile(next.count, next.tooLarge) ? 'file' : 'link')
        }
      },
      (e: unknown) => {
        if (stale) {
          return
        }
        useToasts
          .getState()
          .push({ kind: 'error', title: t`Could not build the link`, body: errorMessage(e) })
        close()
      },
    )
    return () => {
      stale = true
    }
  }, [profileId, keys, game, close, t])
  return (
    <Dialog
      open={profileId !== '' && info !== null}
      onClose={close}
      maxWidth={false}
      transitionDuration={0}
      slotProps={{
        paper: {
          ...paper,
          sx: {
            ...paper.sx,
            bgcolor: 'rgb(36,36,44)',
            border: '1px solid rgba(255,255,255,0.12)',
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
            <TipBanner tip="share">
              {t`A share link names this profile and the Nexus or GitHub files in it, not the archives.`}
            </TipBanner>
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
                <Typography noWrap={true} title={info.name} sx={{ fontSize: 24, fontWeight: 700 }}>
                  {info.name}
                </Typography>
              </Box>
              <CopyModList />
              {tab === 'file' ? (
                <Tooltip title={t`Close`}>
                  <IconButton
                    aria-label={t`Close`}
                    onClick={close}
                    sx={{ alignSelf: 'flex-start' }}
                  >
                    <X size={18} />
                  </IconButton>
                </Tooltip>
              ) : null}
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
            <Box sx={{ overflowY: 'auto', flex: 1, minHeight: 0 }}>
              {tab === 'link' ? <LinkTab info={info} onFile={() => setTab('file')} /> : null}
              {tab === 'file' ? (
                <FileTab info={info} game={game} profileId={profileId} keys={keys} />
              ) : null}
            </Box>
          </Box>
          {tab === 'link' ? <PagePreview info={info} onClose={close} /> : null}
        </Box>
      ) : null}
    </Dialog>
  )
}
