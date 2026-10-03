import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, Chip, Menu, MenuItem } from '@mui/material'
import {
  ArrowDownToLine,
  ChevronDown,
  Clock,
  Download,
  FilterX,
  Search,
  SquareTerminal,
  X,
} from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { State } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import {
  RunCause,
  Runs,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import { useLaunch } from '../launch/store.ts'
import { useLoader } from '../loader/store.ts'
import { onFilterFocus } from '../mods/filterFocus.ts'
import { useProfiles } from '../profiles/store.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { IconAction } from '../shell/IconAction.tsx'
import { TipBanner } from '../tips/TipBanner.tsx'
import { incompatibleSMAPI, isFiltered, modsOf } from './filter.ts'
import { stepHistory } from './history.ts'
import { LevelToggles } from './LevelToggles.tsx'
import { LinkedLog } from './LinkedLog.tsx'
import { LogActions } from './LogActions.tsx'
import { useShownEntries, useVisible } from './logHooks.ts'
import { RunProblemsStrip } from './RunProblems.tsx'
import { RunsPicker } from './RunsPicker.tsx'
import { canSendTo, useConsole } from './store.ts'

function ModPicker() {
  const { t } = useLingui()
  const entries = useShownEntries()
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
        sx={{ height: 34, borderColor: 'var(--mortar-hairline-20)', color: 'var(--mortar-ink)' }}
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
  const input = useRef<HTMLInputElement>(null)
  useEffect(() => onFilterFocus(() => input.current?.focus()), [])
  return (
    <Box
      component="label"
      sx={{
        flex: '1 1 120px',
        minWidth: 120,
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        height: 34,
        px: 1.25,
        bgcolor: 'var(--mortar-overlay-30)',
        border: '1px solid var(--mortar-hairline-15)',
        borderRadius: '6px',
        color: 'var(--mortar-ink-dim-92)',
      }}
    >
      <Search size={14} aria-hidden={true} />
      <Box
        component="input"
        ref={input}
        aria-label={t`Search the log`}
        placeholder={t`Search the log`}
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        sx={{
          flexGrow: 1,
          minWidth: 0,
          bgcolor: 'transparent',
          border: 0,
          color: 'var(--mortar-ink)',
          font: 'inherit',
          fontSize: 13,
          outline: 'none',
        }}
      />
    </Box>
  )
}

const NO_HISTORY: string[] = []
const MONO = 'ui-monospace, "SFMono-Regular", Menlo, Consolas, monospace'

function CauseBanner({ game, profile, run }: { game: string; profile: string; run: string }) {
  const { t } = useLingui()
  const [cause, setCause] = useState<Awaited<ReturnType<typeof RunCause>> | null>(null)
  useEffect(() => {
    if (!run) {
      setCause(null)
      return
    }
    RunCause(game, profile, run).then(setCause, () => setCause(null))
  }, [game, profile, run])
  return cause?.modName ? (
    <Box
      sx={{ mx: 2, mb: 1, px: 1.5, py: 1, bgcolor: 'rgba(180,80,70,0.25)', borderRadius: '6px' }}
    >
      <strong>{t`Caused by ${cause.modName}`}</strong> {t`·`} {cause.detail}
    </Box>
  ) : null
}

function CommandLine({ game }: { game: string }) {
  const { t } = useLingui()
  const openId = useProfiles((s) => s.openId)
  const viewingRun = useConsole((s) => s.viewingRun)
  const running = useLaunch((s) => canSendTo(s.status, game, openId)) && viewingRun === ''
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
  // With no game to send to, the input only takes room from the log.
  if (!running) {
    return null
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
        mt: -0.5,
        mb: 1.5,
        px: 1.5,
        flexShrink: 0,
        bgcolor: 'var(--mortar-overlay-50)',
        border: '1px solid var(--mortar-hairline-15)',
        borderRadius: '6px',
        fontFamily: MONO,
        fontSize: 13,
        color: running ? 'var(--mortar-ink)' : 'var(--mortar-ink-dim-60)',
      }}
    >
      <span aria-hidden={true}>{'>'}</span>
      <Box
        component="input"
        aria-label={t`Console command`}
        placeholder={t`Type a command, for example help`}
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

function ReinstallLoader({ game }: { game: string }) {
  const { t } = useLingui()
  const install = useLoader((s) => s.install)
  const installing = useLoader((s) => s.installing)
  const pending = useLoader((s) => s.pending)
  const playing = useLaunch(
    (s) =>
      s.starting ||
      (s.status?.game === game &&
        (s.status.state === State.Launching || s.status.state === State.Running)),
  )
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'flex-end',
        px: 2,
        pb: 1,
        flexShrink: 0,
      }}
    >
      <Button
        variant="contained"
        size="small"
        startIcon={<Download size={16} />}
        disabled={pending || installing || playing}
        onClick={() => install(game)}
        sx={{ whiteSpace: 'nowrap', flexShrink: 0 }}
      >
        {t`Reinstall`}
      </Button>
    </Box>
  )
}

// useHasRuns reports whether the profile has any recorded run, re-reading once the game stops so a run that just
// ended counts. Until it knows, it assumes runs exist so the log never flashes the empty state.
function useHasRuns(game: string, profile: string, running: boolean): boolean {
  const [hasRuns, setHasRuns] = useState(true)
  useEffect(() => {
    if (running) {
      return
    }
    let live = true
    Runs(game, profile).then(
      (list) => {
        if (live) {
          setHasRuns((list ?? []).length > 0)
        }
      },
      () => {
        if (live) {
          setHasRuns(true)
        }
      },
    )
    return () => {
      live = false
    }
  }, [game, profile, running])
  return hasRuns
}

// ConsoleEmpty stands in for the log until the profile has ever run, so a first visit is not an empty black box.
function ConsoleEmpty() {
  const { t } = useLingui()
  return (
    <EmptyState
      icon={<SquareTerminal size={40} aria-hidden={true} />}
      title={t`No game output yet`}
    >
      {t`Play this profile and SMAPI's log shows here as it runs. Every run is kept, so you can look back at it later.`}
    </EmptyState>
  )
}

export function ConsoleTab({ game }: { game: string }) {
  const { t } = useLingui()
  const entries = useShownEntries()
  const filters = useConsole((s) => s.filters)
  const timestamps = useConsole((s) => s.timestamps)
  const follow = useConsole((s) => s.follow)
  const jump = useConsole((s) => s.jump)
  const { load, setMods, clearFilters, setTimestamps, setFollow } = useConsole.getState()
  const shown = useConsole((s) => s.shown)
  const openId = useProfiles((s) => s.openId)
  const viewingRun = useConsole((s) => s.viewingRun)
  // The live log belongs to the open profile; only a past run (e.g. opened from a crash summary) names its own.
  const target = viewingRun
    ? { game: shown.game || game, profile: shown.profile }
    : { game, profile: openId }
  const rows = useVisible()
  const offerReinstall = entries.some((e) => incompatibleSMAPI(e.message))
  // Launching another profile resets the log to it; this one's history is read again once that launch settles.
  const launchingOther = useLaunch(
    (s) =>
      s.status?.state === State.Launching &&
      s.status.game === game &&
      s.status.profile !== target.profile,
  )
  const [loaded, setLoaded] = useState(false)
  useEffect(() => {
    if (launchingOther) {
      return
    }
    let live = true
    setLoaded(false)
    load(target.game, target.profile).then(() => {
      if (live) {
        setLoaded(true)
      }
    })
    return () => {
      live = false
    }
  }, [target.game, target.profile, launchingOther, load])
  const total = entries.length
  const running = useLaunch((s) => canSendTo(s.status, game, openId))
  const hasRuns = useHasRuns(game, openId, running)
  if (loaded && total === 0 && viewingRun === '' && !running && !hasRuns) {
    return <ConsoleEmpty />
  }
  let empty: string | null = null
  if (loaded && total === 0) {
    empty = t`The console fills when the game runs.`
  } else if (loaded && rows.length === 0) {
    empty = t`No lines match the filters.`
  }
  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
      <TipBanner tip="console">
        {t`Filter by level or mod, and use Share log to upload this run to smapi.io.`}
      </TipBanner>
      <Box
        sx={{
          display: 'flex',
          flexWrap: 'nowrap',
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
        <RunsPicker game={game} />
        <IconAction
          label={t`Show times`}
          icon={<Clock size={16} />}
          pressed={timestamps}
          onClick={() => setTimestamps(!timestamps)}
        />
        <IconAction
          label={t`Follow new lines`}
          icon={<ArrowDownToLine size={16} />}
          pressed={follow}
          onClick={() => setFollow(!follow)}
        />
        {isFiltered(filters) ? (
          <IconAction
            label={t`Clear filters`}
            icon={<FilterX size={16} />}
            onClick={clearFilters}
          />
        ) : null}
        <Box sx={{ ml: 'auto', display: 'flex', gap: 0.75 }}>
          <LogActions game={game} />
        </Box>
      </Box>
      {filters.mods.length > 0 ? (
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
              sx={{ bgcolor: 'var(--mortar-hairline)', fontSize: 13 }}
            />
          ))}
        </Box>
      ) : null}
      <CauseBanner game={target.game} profile={target.profile} run={viewingRun} />
      <RunProblemsStrip game={target.game} profile={target.profile} run={viewingRun} />
      {offerReinstall ? <ReinstallLoader game={game} /> : null}
      <LinkedLog
        game={game}
        rows={rows}
        timestamps={timestamps}
        follow={follow}
        jump={jump}
        empty={empty}
        onUnfollow={() => setFollow(false)}
      />
      <CommandLine game={game} />
    </Box>
  )
}
