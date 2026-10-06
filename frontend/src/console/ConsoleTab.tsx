import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, Chip, Menu, MenuItem } from '@mui/material'
import {
  ArrowDownToLine,
  ChevronDown,
  Clock,
  Download,
  FilterX,
  SquareTerminal,
  X,
} from 'lucide-react'
import { type ReactNode, useEffect, useMemo, useRef, useState } from 'react'
import { State } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { RunCause } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { listNames } from '../i18n/list.ts'
import { useGameBusy, useLaunch } from '../launch/store.ts'
import { useLoader } from '../loader/store.ts'
import { onFilterFocus } from '../mods/filterFocus.ts'
import { useProfileLoader, useProfiles } from '../profiles/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { IconAction } from '../shell/IconAction.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { MONO, PAD_FOCUS } from '../theme/theme.ts'
import { TipBanner } from '../tips/TipBanner.tsx'
import { hiddenBy, incompatibleSMAPI, isFiltered, modsOf } from './filter.ts'
import { stepHistory } from './history.ts'
import { LevelToggles } from './LevelToggles.tsx'
import { LinkedLog } from './LinkedLog.tsx'
import { LogActions } from './LogActions.tsx'
import { useLevelNames, useShownEntries, useVisible } from './logHooks.ts'
import { RunProblemsStrip } from './RunProblems.tsx'
import { RunsPicker } from './RunsPicker.tsx'
import { canSendTo, useConsole } from './store.ts'
import { useConsoleEmpty } from './useConsoleEmpty.ts'

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
        aria-haspopup="menu"
        aria-expanded={anchor !== null}
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
              role="menuitemcheckbox"
              aria-checked={on}
              onClick={() => setMods(on ? picked.filter((m) => m !== mod) : [...picked, mod])}
            >
              <Checkbox
                size="small"
                checked={on}
                tabIndex={-1}
                disableRipple={true}
                slotProps={{ input: { 'aria-hidden': true } }}
              />
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
    <SearchField
      value={search}
      onChange={setSearch}
      label={t`Search the log`}
      inputRef={input}
      sx={{ flex: '1 1 120px', minWidth: 120 }}
    />
  )
}

const NO_HISTORY: string[] = []

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
  const takesCommands = useProfileLoader()?.commands === true
  const running =
    useLaunch((s) => canSendTo(s.status, game, openId)) && viewingRun === '' && takesCommands
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
  // With no game to send to, or one that takes no commands, the input only takes room from the log.
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
        // A light-mode tint over the wallpaper reads as grey; an opaque paper well stays legible.
        bgcolor: (theme) =>
          theme.palette.mode === 'light'
            ? theme.palette.background.paper
            : 'var(--mortar-overlay-50)',
        border: '1px solid var(--mortar-hairline-15)',
        borderRadius: '6px',
        [`&:has(:focus-visible), ${PAD_FOCUS} &:has(:focus)`]: {
          outline: '2px solid',
          outlineColor: 'primary.main',
          outlineOffset: '2px',
        },
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
  const playing = useGameBusy(game)
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
      <DisabledReason
        title={t`Stop the game before reinstalling.`}
        disabled={pending || installing || playing}
      >
        <Button
          variant="contained"
          size="small"
          startIcon={<Download size={16} />}
          disabled={pending || installing || playing}
          onClick={() => install(game)}
          sx={{ flexShrink: 0 }}
        >
          {t`Reinstall`}
        </Button>
      </DisabledReason>
    </Box>
  )
}

// ConsoleTip points at the way out for this run's log: Share log for a loader smapi.io reads, else Save and Copy.
function ConsoleTip() {
  const { t } = useLingui()
  const shares = useProfileLoader()?.share === true
  return (
    <TipBanner tip="console">
      {shares
        ? t`Filter by level or mod, and use Share log to upload this run to smapi.io.`
        : t`Filter by level or mod, and use Save log or Copy log to pass this run on.`}
    </TipBanner>
  )
}

// ConsoleEmpty stands in for the log until the profile has ever run, so a first visit is not an empty black box.
function ConsoleEmpty() {
  const { t } = useLingui()
  return (
    <EmptyState
      icon={<SquareTerminal size={40} aria-hidden={true} />}
      title={t`No game output yet`}
    >
      {t`Play this profile and the game's log shows here as it runs. Every run is kept.`}
    </EmptyState>
  )
}

// Every line is filtered out: say how many and by what, so a quiet level chip does not read as an empty log.
function FilteredEmpty() {
  const { t } = useLingui()
  const entries = useShownEntries()
  const filters = useConsole((s) => s.filters)
  const showAll = useConsole((s) => s.showAll)
  const names = useLevelNames()
  const hidden = hiddenBy(entries, filters)
  const reasons: string[] = []
  if (hidden.levels.length > 0) {
    const levels = listNames(
      hidden.levels.map((l) => names[l]),
      hidden.levels.length,
    )
    reasons.push(
      plural(hidden.levels.length, {
        one: `the ${levels} level is off`,
        other: `the ${levels} levels are off`,
      }),
    )
  }
  if (hidden.search !== '') {
    reasons.push(t`the search is “${hidden.search}”`)
  }
  if (hidden.mods.length > 0) {
    const mods = listNames(hidden.mods)
    reasons.push(t`only ${mods} is shown`)
  }
  if (hidden.excludeMods.length > 0) {
    const mods = listNames(hidden.excludeMods)
    reasons.push(t`${mods} is hidden`)
  }
  const why = listNames(reasons, reasons.length)
  return (
    <>
      {plural(hidden.count, {
        one: `# line is hidden by the filters: ${why}.`,
        other: `# lines are hidden by the filters: ${why}.`,
      })}{' '}
      <Button size="small" onClick={showAll} sx={{ verticalAlign: 'baseline' }}>
        {plural(hidden.count, { one: 'Show it', other: 'Show all # lines' })}
      </Button>
    </>
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
  const consoleEmpty = useConsoleEmpty(game)
  if (loaded && consoleEmpty) {
    return <ConsoleEmpty />
  }
  let empty: ReactNode = null
  if (loaded && total === 0) {
    empty = t`The console fills when the game runs.`
  } else if (loaded && rows.length === 0) {
    empty = <FilteredEmpty />
  }
  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
      <ConsoleTip />
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
