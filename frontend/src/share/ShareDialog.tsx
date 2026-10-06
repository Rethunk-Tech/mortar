import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  ButtonGroup,
  Divider,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Typography,
} from '@mui/material'
import { Clipboard } from '@wailsio/runtime'
import { Check, ChevronDown, Copy, FileText, List, MessageSquare, Save, Type } from 'lucide-react'
import { useState } from 'react'
import type { Saved } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/models.ts'
import { SaveFile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/service.ts'
import { Logo } from '../brand/Logo.tsx'
import { SendDialog } from '../lan/SendDialog.tsx'
import { heading } from '../mods/paper.ts'
import { useProfiles } from '../profiles/store.ts'
import { calloutFill, calloutLine } from '../theme/callout.ts'
import { MONO } from '../theme/theme.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { IncludeOptions } from './IncludeOptions.tsx'
import { type MeterLevel, meter, type ShownInfo, suggestFile } from './logic.ts'
import { formatModList, listItems, type ModListFormat } from './modList.ts'
import { SharedMods } from './SharedMods.tsx'
import { ShareShell } from './ShareShell.tsx'
import { type ShareInclude, toShareInclude } from './shareDefaults.ts'
import { useShareDialog } from './store.ts'
import { useShareBuild } from './useShareBuild.ts'

const PERCENT = 100
const PAGE_HOST = 'mortar.rethunk.tech'

const LEVEL_TEXT: Record<MeterLevel, string> = {
  ok: 'success.main',
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
          bgcolor: 'var(--mortar-hairline)',
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

function PagePreview({ info }: { info: ShownInfo }) {
  const { t } = useLingui()
  const art = useProfiles((s) => s.game?.artUrl)
  const gameName = useProfiles((s) => s.game?.name ?? '')
  const { count } = info
  return (
    <Box
      component="aside"
      aria-label={t`What the person opening the link sees`}
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: 1.5,
        p: 2.5,
        bgcolor: 'var(--mortar-console-90)',
        borderLeft: '1px solid var(--mortar-hairline-muted)',
      }}
    >
      <Typography sx={{ ...heading, minHeight: 40, display: 'flex', alignItems: 'center' }}>
        {t`What they see`}
      </Typography>
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
        <Typography sx={{ fontSize: 13, color: 'inherit' }}>
          {plural(count, {
            one: `${{ name: gameName }} · # mod`,
            other: `${{ name: gameName }} · # mods`,
          })}
        </Typography>
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

function LinkTab({
  info,
  onFile,
  include,
  onInclude,
}: {
  info: ShownInfo
  onFile: () => void
  include: ShareInclude
  onInclude: (next: ShareInclude) => void
}) {
  const { t } = useLingui()
  const gameName = useProfiles((s) => s.game?.name ?? '')
  const copy = (text: string, done: string) => {
    Clipboard.SetText(text).then(
      () => useToasts.getState().push({ kind: 'success', title: done }),
      reportUnexpected,
    )
  }
  const { count } = info
  const message = plural(count, {
    one: `Try my Mortar profile "${info.name}" for ${gameName} (# mod): ${info.web}`,
    other: `Try my Mortar profile "${info.name}" for ${gameName} (# mods): ${info.web}`,
  })
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
                  bgcolor: 'var(--mortar-overlay-45)',
                  border: '1px solid var(--mortar-hairline-15)',
                  borderRadius: '6px',
                  fontFamily: MONO,
                  fontSize: 13,
                  userSelect: 'text',
                }}
              >
                <Typography noWrap={true} title={info.web} sx={{ font: 'inherit' }}>
                  {info.web}
                </Typography>
              </Box>
              <Button
                variant="contained"
                startIcon={<Copy size={16} />}
                disabled={count === 0}
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
                disabled={count === 0}
                onClick={() => copy(message, t`Message copied`)}
                sx={{ height: 40, px: '14px', fontSize: 14 }}
              >
                {t`Copy as a message`}
              </Button>
            </Box>
          </>
        )}
        <IncludeOptions value={include} onChange={onInclude} file={false} />
        {large ? (
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: 1.5,
              p: 1.5,
              bgcolor: calloutFill('warning'),
              border: '1px solid',
              borderColor: calloutLine('warning'),
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
  const [partsAnchor, setPartsAnchor] = useState<HTMLElement | null>(null)
  const items = listItems(profile, keys)
  const labels = { enabled: t`Enabled`, disabled: t`Disabled` }
  const parts = formatModList(format, items, labels)
  const options: { id: ModListFormat; label: string; icon: typeof FileText }[] = [
    { id: 'markdown', label: t`Markdown`, icon: FileText },
    { id: 'plain', label: t`Plain text`, icon: Type },
    { id: 'discord', label: t`Discord`, icon: MessageSquare },
  ]
  return (
    <>
      <ButtonGroup variant="outlined" color="inherit" sx={{ mr: 5 }}>
        <Button
          startIcon={<List size={16} />}
          aria-haspopup={parts.length > 1 ? 'menu' : undefined}
          aria-expanded={parts.length > 1 ? partsAnchor !== null : undefined}
          onClick={(e) => {
            if (parts.length > 1) {
              setPartsAnchor(e.currentTarget)
              return
            }
            copyText(parts[0]?.text ?? '', t`Mod list copied`)
          }}
          sx={{ height: 40, px: '14px', fontSize: 14 }}
        >
          {parts.length > 1
            ? plural(parts.length, {
                one: 'Copy mod list (# part)',
                other: 'Copy mod list (# parts)',
              })
            : t`Copy mod list`}
        </Button>
        <Button
          aria-label={t`Mod list format`}
          endIcon={<ChevronDown size={14} />}
          aria-haspopup="menu"
          aria-expanded={anchor !== null}
          onClick={(e) => setAnchor(e.currentTarget)}
          sx={{ height: 40, px: '14px', fontSize: 14 }}
        >
          {options.find((o) => o.id === format)?.label}
        </Button>
      </ButtonGroup>
      <Menu open={Boolean(anchor)} anchorEl={anchor} onClose={() => setAnchor(null)}>
        {options.map((o) => {
          const Icon = o.icon
          return (
            <MenuItem
              key={o.id}
              role="menuitemradio"
              aria-checked={o.id === format}
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
      <Menu open={Boolean(partsAnchor)} anchorEl={partsAnchor} onClose={() => setPartsAnchor(null)}>
        {parts.map((part) => (
          <MenuItem
            key={part.id}
            onClick={() => {
              copyText(part.text, t`Part ${part.n} copied`)
              setPartsAnchor(null)
            }}
          >
            <ListItemIcon sx={{ color: 'inherit' }}>
              <Copy size={16} />
            </ListItemIcon>
            <ListItemText>{t`Copy part ${part.n} of ${parts.length}`}</ListItemText>
          </MenuItem>
        ))}
      </Menu>
    </>
  )
}

function FileTab({
  info,
  game,
  profileId,
  keys,
  include,
  onInclude,
}: {
  info: ShownInfo
  game: string
  profileId: string
  keys: string[]
  include: ShareInclude
  onInclude: (next: ShareInclude) => void
}) {
  const { t } = useLingui()
  const [saved, setSaved] = useState<Saved | null>(null)
  const [busy, run] = usePending()
  const save = () => {
    run(
      async () => {
        const result = await SaveFile(game, profileId, keys, toShareInclude(include))
        if (result.path) {
          setSaved(result)
          useToasts.getState().push({ kind: 'success', title: t`File saved` })
        }
      },
      { errorTitle: t`Could not save the file` },
    )
  }
  const skipped = saved?.skipped ?? []
  return (
    <>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.25, p: '18px 24px' }}>
        <Typography sx={{ fontSize: 14, lineHeight: 1.5 }}>
          {t`A .mortar file holds the link's mods plus their settings files. Send it; they open it from Mortar's Import.`}
        </Typography>
        <IncludeOptions value={include} onChange={onInclude} file={true} />
        <Box>
          <Button
            variant="contained"
            startIcon={<Save size={16} />}
            disabled={busy || info.count === 0}
            onClick={save}
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
  const art = useProfiles((s) => s.game?.artUrl)
  const [sendNearby, setSendNearby] = useState(false)
  const built = useShareBuild()
  const { profileId, keys, close, game, tab, setTab, info, include, setInclude } = built
  return (
    <>
      <ShareShell
        art={art}
        onSendNearby={setSendNearby}
        profileId={profileId}
        close={close}
        tab={tab}
        setTab={setTab}
        info={info}
        headerExtra={<CopyModList />}
        preview={tab === 'link' && info ? <PagePreview info={info} /> : null}
      >
        {tab === 'link' && info ? (
          <LinkTab
            info={info}
            onFile={() => setTab('file')}
            include={include}
            onInclude={setInclude}
          />
        ) : null}
        {tab === 'file' && info ? (
          <FileTab
            info={info}
            game={game}
            profileId={profileId}
            keys={keys}
            include={include}
            onInclude={setInclude}
          />
        ) : null}
      </ShareShell>
      <SendDialog
        open={sendNearby}
        game={game}
        profileId={profileId}
        onClose={() => setSendNearby(false)}
      />
    </>
  )
}
