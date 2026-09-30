import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  ButtonBase,
  Checkbox,
  Chip,
  FormControlLabel,
  Menu,
  MenuItem,
  Typography,
} from '@mui/material'
import { Clipboard } from '@wailsio/runtime'
import { ChevronDown, LifeBuoy, Search, X } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Level } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import { State } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { useLaunch } from '../launch/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import {
  countByLevel,
  firstError,
  formatAll,
  isFiltered,
  LEVELS,
  modsOf,
  visible,
} from './filter.ts'
import { stepHistory } from './history.ts'
import { useConsole } from './store.ts'
import { VirtualLog } from './VirtualLog.tsx'

const dots: Record<Level, string> = {
  [Level.$zero]: 'transparent',
  [Level.Trace]: '#9a9aa6',
  [Level.Debug]: '#b4b4c0',
  [Level.Info]: '#ececf0',
  [Level.Warn]: '#F3B416',
  [Level.Error]: '#ff6b5f',
  [Level.Alert]: '#c792ea',
}

// A launch of another profile takes the log over; its lines are not this profile's to show.
function useVisible() {
  const entries = useConsole((s) => s.entries)
  const filters = useConsole((s) => s.filters)
  const mine = useConsole((s) => s.shown.profile) === useProfiles((s) => s.openId)
  return useMemo(() => (mine ? visible(entries, filters) : []), [mine, entries, filters])
}

function LevelToggles() {
  const { t } = useLingui()
  const entries = useConsole((s) => s.entries)
  const on = useConsole((s) => s.filters.levels)
  const toggle = useConsole((s) => s.toggleLevel)
  const counts = useMemo(() => countByLevel(entries), [entries])
  const names: Record<Level, string> = {
    [Level.$zero]: '',
    [Level.Trace]: t`Trace`,
    [Level.Debug]: t`Debug`,
    [Level.Info]: t`Info`,
    [Level.Warn]: t`Warn`,
    [Level.Error]: t`Error`,
    [Level.Alert]: t`Alert`,
  }
  return (
    <Box
      role="group"
      aria-label={t`Levels`}
      sx={{
        display: 'flex',
        p: '3px',
        gap: '2px',
        bgcolor: 'rgba(0,0,0,0.3)',
        borderRadius: '8px',
      }}
    >
      {LEVELS.map((level) => {
        const pressed = on.includes(level)
        return (
          <ButtonBase
            key={level}
            aria-pressed={pressed}
            onClick={() => toggle(level)}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              height: 28,
              px: 1.25,
              borderRadius: '6px',
              fontSize: 13,
              whiteSpace: 'nowrap',
              bgcolor: pressed ? 'rgba(255,255,255,0.14)' : 'transparent',
              color: pressed ? '#ffffff' : 'rgba(210,210,215,0.85)',
              '&:hover': { bgcolor: pressed ? 'rgba(255,255,255,0.18)' : 'rgba(255,255,255,0.08)' },
            }}
          >
            <Box sx={{ width: 8, height: 8, borderRadius: '4px', bgcolor: dots[level] }} />
            {names[level]}
            <Box component="span" sx={{ opacity: 0.75 }}>
              {counts.get(level) ?? 0}
            </Box>
          </ButtonBase>
        )
      })}
    </Box>
  )
}

function ModPicker() {
  const { t } = useLingui()
  const entries = useConsole((s) => s.entries)
  const picked = useConsole((s) => s.filters.mods)
  const setMods = useConsole((s) => s.setMods)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const mods = useMemo(() => modsOf(entries), [entries])
  return (
    <>
      <Button
        variant="outlined"
        color="inherit"
        endIcon={<ChevronDown size={12} />}
        onClick={(e) => setAnchor(e.currentTarget)}
        sx={{ height: 34, borderColor: 'rgba(255,255,255,0.2)', color: '#ffffff' }}
      >
        {t`Mods`}
      </Button>
      <Menu
        open={anchor !== null}
        anchorEl={anchor}
        onClose={() => setAnchor(null)}
        slotProps={{ paper: { sx: { maxHeight: 360, minWidth: 220 } } }}
      >
        {mods.length === 0 ? (
          <MenuItem disabled={true}>{t`No mods have logged yet`}</MenuItem>
        ) : null}
        {mods.map((mod) => {
          const on = picked.includes(mod)
          return (
            <MenuItem
              key={mod}
              dense={true}
              onClick={() => setMods(on ? picked.filter((m) => m !== mod) : [...picked, mod])}
            >
              <Checkbox size="small" checked={on} tabIndex={-1} disableRipple={true} />
              {mod}
            </MenuItem>
          )
        })}
      </Menu>
    </>
  )
}

function SearchBox() {
  const { t } = useLingui()
  const search = useConsole((s) => s.filters.search)
  const setSearch = useConsole((s) => s.setSearch)
  return (
    <Box
      component="label"
      sx={{
        width: 240,
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        height: 34,
        px: 1.25,
        bgcolor: 'rgba(0,0,0,0.3)',
        border: '1px solid rgba(255,255,255,0.15)',
        borderRadius: '6px',
        color: 'rgba(210,210,215,0.92)',
      }}
    >
      <Search size={14} aria-hidden={true} />
      <Box
        component="input"
        aria-label={t`Search the log`}
        placeholder={t`Search the log`}
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        sx={{
          flexGrow: 1,
          minWidth: 0,
          bgcolor: 'transparent',
          border: 0,
          color: '#ffffff',
          font: 'inherit',
          fontSize: 13,
          outline: 'none',
        }}
      />
    </Box>
  )
}

function Toggle({
  label,
  checked,
  onChange,
}: {
  label: string
  checked: boolean
  onChange: (on: boolean) => void
}) {
  return (
    <FormControlLabel
      control={
        <Checkbox size="small" checked={checked} onChange={(e) => onChange(e.target.checked)} />
      }
      label={label}
      sx={{ m: 0, whiteSpace: 'nowrap', '& .MuiFormControlLabel-label': { fontSize: 13 } }}
    />
  )
}

const NO_HISTORY: string[] = []
const MONO = 'ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace'

function CommandLine({ game }: { game: string }) {
  const { t } = useLingui()
  const running = useLaunch((s) => s.status?.state === State.Running && s.status.game === game)
  const history = useConsole((s) => s.history[game] ?? NO_HISTORY)
  const send = useConsole((s) => s.send)
  const [text, setText] = useState('')
  // The history entry shown, or null while the user types their own line.
  const [cursor, setCursor] = useState<number | null>(null)
  const draft = useRef('')
  // Enter pressed again before Send answers must not run the command twice.
  const sending = useRef(false)
  const browse = (dir: -1 | 1) => {
    if (cursor === null && dir === 1) {
      return
    }
    const next = stepHistory(history, cursor ?? history.length, dir)
    if (cursor === null) {
      draft.current = text
    }
    if (next === history.length) {
      setCursor(null)
      setText(draft.current)
    } else {
      setCursor(next)
      setText(history[next] ?? '')
    }
  }
  const submit = () => {
    if (text.trim() === '' || sending.current) {
      return
    }
    sending.current = true
    send(game, text)
      .then((sent) => {
        if (sent) {
          setText('')
          setCursor(null)
        }
      })
      .finally(() => {
        sending.current = false
      })
  }
  return (
    <Box
      component="label"
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        height: 38,
        mx: 2,
        mb: 2,
        px: 1.5,
        flexShrink: 0,
        bgcolor: 'rgba(0,0,0,0.5)',
        border: '1px solid rgba(255,255,255,0.15)',
        borderRadius: '6px',
        fontFamily: MONO,
        fontSize: 13,
        color: running ? '#ffffff' : 'rgba(210,210,215,0.6)',
      }}
    >
      <span aria-hidden={true}>{'>'}</span>
      <Box
        component="input"
        aria-label={t`Console command`}
        placeholder={
          running ? t`Type a command, for example help` : t`Start the game to run commands`
        }
        disabled={!running}
        value={text}
        spellCheck={false}
        autoComplete="off"
        onChange={(e) => {
          setText(e.target.value)
          setCursor(null)
        }}
        onKeyDown={(e) => {
          if (e.nativeEvent.isComposing) {
            return
          }
          if (e.key === 'Enter') {
            submit()
          } else if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
            e.preventDefault()
            browse(e.key === 'ArrowUp' ? -1 : 1)
          }
        }}
        sx={{
          flexGrow: 1,
          minWidth: 0,
          bgcolor: 'transparent',
          border: 0,
          color: 'inherit',
          font: 'inherit',
          outline: 'none',
        }}
      />
    </Box>
  )
}

const actionSx = {
  height: 32,
  whiteSpace: 'nowrap',
  borderColor: 'rgba(255,255,255,0.22)',
  color: '#ffffff',
} as const

export function ConsoleTab({ game }: { game: string }) {
  const { t } = useLingui()
  const entries = useConsole((s) => s.entries)
  const filters = useConsole((s) => s.filters)
  const timestamps = useConsole((s) => s.timestamps)
  const follow = useConsole((s) => s.follow)
  const jump = useConsole((s) => s.jump)
  const { load, setMods, clearFilters, setTimestamps, setFollow } = useConsole.getState()
  const profile = useProfiles((s) => s.openId)
  const rows = useVisible()
  // Launching another profile resets the log to it; this one's history is read again once that launch settles.
  const launchingOther = useLaunch(
    (s) => s.status?.state === State.Launching && s.status.game === game && s.status.profile !== profile,
  )
  const [loaded, setLoaded] = useState(false)
  useEffect(() => {
    if (launchingOther) {
      return
    }
    let live = true
    setLoaded(false)
    load(game, profile).then(() => {
      if (live) {
        setLoaded(true)
      }
    })
    return () => {
      live = false
    }
  }, [game, profile, launchingOther, load])
  const total = entries.length
  let empty: string | null = null
  if (loaded && total === 0) {
    empty = t`The console fills when the game runs.`
  } else if (loaded && rows.length === 0) {
    empty = t`No lines match the filters.`
  }
  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
      <Box
        sx={{
          display: 'flex',
          flexWrap: 'wrap',
          alignItems: 'center',
          gap: 1,
          px: 2,
          pt: 1.25,
          pb: 0.75,
          flexShrink: 0,
        }}
      >
        <SearchBox />
        <LevelToggles />
        <ModPicker />
        <Box sx={{ flexGrow: 1 }} />
        <Toggle label={t`Times`} checked={timestamps} onChange={setTimestamps} />
        <Toggle label={t`Follow`} checked={follow} onChange={setFollow} />
      </Box>
      {isFiltered(filters) ? (
        <Box
          sx={{
            display: 'flex',
            flexWrap: 'wrap',
            alignItems: 'center',
            gap: 1,
            px: 2,
            pb: 1,
            flexShrink: 0,
            fontSize: 13,
            whiteSpace: 'nowrap',
          }}
        >
          {filters.mods.map((mod) => (
            <Chip
              key={mod}
              size="small"
              label={t`Mod: ${mod}`}
              onDelete={() => setMods(filters.mods.filter((m) => m !== mod))}
              deleteIcon={<X size={14} aria-label={t`Remove filter ${mod}`} />}
              sx={{ bgcolor: 'rgba(255,255,255,0.1)', fontSize: 13 }}
            />
          ))}
          <Button
            size="small"
            variant="text"
            color="inherit"
            onClick={clearFilters}
            sx={{ minWidth: 0, textDecoration: 'underline' }}
          >
            {t`Clear filters`}
          </Button>
        </Box>
      ) : null}
      <Box
        sx={{
          flex: 1,
          minHeight: 0,
          mx: 2,
          mb: 1,
          bgcolor: 'rgba(0,0,0,0.5)',
          borderRadius: '6px',
          fontFamily: MONO,
          fontSize: 13,
          lineHeight: '23px',
          overflow: 'hidden',
        }}
      >
        {empty ? (
          <Typography sx={{ p: 2, font: 'inherit', color: 'text.secondary' }}>{empty}</Typography>
        ) : (
          <VirtualLog
            rows={rows}
            timestamps={timestamps}
            follow={follow}
            onUnfollow={() => setFollow(false)}
            jump={jump}
          />
        )}
      </Box>
      <CommandLine game={game} />
    </Box>
  )
}

export function LogActions() {
  const { t } = useLingui()
  const rows = useVisible()
  const { clear, jumpTo, setHelping } = useConsole.getState()
  const firstErr = firstError(rows)
  return (
    <Box sx={{ display: 'flex', gap: 1 }}>
      <Button
        variant="outlined"
        color="inherit"
        disabled={firstErr < 0}
        onClick={() => jumpTo(firstErr)}
        sx={actionSx}
      >
        {t`Jump to first error`}
      </Button>
      <Button
        variant="outlined"
        color="inherit"
        disabled={rows.length === 0}
        onClick={clear}
        sx={actionSx}
      >
        {t`Clear`}
      </Button>
      <Button
        variant="outlined"
        color="inherit"
        disabled={rows.length === 0}
        onClick={() => {
          Clipboard.SetText(formatAll(rows)).then(
            () => useToasts.getState().push({ kind: 'success', title: t`Log copied` }),
            reportUnexpected,
          )
        }}
        sx={actionSx}
      >
        {t`Copy`}
      </Button>
      <Button
        variant="outlined"
        color="inherit"
        startIcon={<LifeBuoy size={16} />}
        onClick={() => setHelping(true)}
        sx={actionSx}
      >
        {t`Get help`}
      </Button>
    </Box>
  )
}
