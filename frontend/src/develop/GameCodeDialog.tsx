import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  IconButton,
  MenuItem,
  ToggleButton,
  ToggleButtonGroup,
  Typography,
} from '@mui/material'
import { Code, Copy, FileSearch } from 'lucide-react'
import { type ReactNode, useEffect, useState } from 'react'
import type { Usage } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  Assemblies,
  Decompile,
  GameCode,
  HasDecompiler,
  InstallDecompiler,
  Usages,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import type { Member } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/dotnet/models.ts'
import { ILSPY, MONOMOD } from '../brand/productNames.ts'
import { copyText } from '../share/copyText.ts'
import { AppDialog, ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { SelectField } from '../shell/dialog/FormDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { Tooltip } from '../shell/Tooltip.tsx'
import { useLoaded } from '../shell/useLoaded.ts'
import { space } from '../theme/density.ts'
import { MONO } from '../theme/theme.ts'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { CodeView } from './CodeView.tsx'
import { hookSnippet } from './hook.ts'

// A constructor's decompiled declaration is named after its type, which a search for ".ctor" cannot find.
const CONSTRUCTOR_NAME = /^\.c?ctor$/
const TYPE_PARTS = /[.+]/

// Typing settles for this long before the game's members are searched.
const SEARCH_DELAY_MS = 200

function useHits(game: string, dir: string, assembly: string, query: string) {
  const [hits, setHits] = useState<Member[]>([])
  const [error, setError] = useState('')
  useEffect(() => {
    if (query.trim().length < 2) {
      setHits([])
      return
    }
    const timer = setTimeout(() => {
      GameCode(game, dir, assembly, query)
        .then((found) => {
          setHits(found ?? [])
          setError('')
        })
        .catch((e: unknown) => setError(e instanceof Error ? e.message : String(e)))
    }, SEARCH_DELAY_MS)
    return () => clearTimeout(timer)
  }, [game, dir, assembly, query])
  return { hits, error }
}

function HitRow({ hit, selected, onPick }: { hit: Member; selected: boolean; onPick: () => void }) {
  const { t } = useLingui()
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 0.5,
        px: 1,
        borderRadius: '6px',
        bgcolor: selected ? 'var(--mortar-raised)' : undefined,
      }}
    >
      <Box
        component="button"
        type="button"
        onClick={onPick}
        title={`${hit.type}::${hit.method}`}
        sx={{
          all: 'unset',
          flex: 1,
          minWidth: 0,
          py: 0.5,
          cursor: 'pointer',
        }}
      >
        <Typography noWrap={true} sx={{ fontFamily: MONO, fontSize: 'small.fontSize' }}>
          {hit.kind === 'field' ? hit.method : `${hit.method}(${(hit.params ?? []).join(', ')})`}
        </Typography>
        <Typography noWrap={true} sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
          {hit.type}
        </Typography>
      </Box>
      {hit.kind === 'field' ? null : (
        <Tooltip title={t`Copy a ${{ library: MONOMOD }} hook`}>
          <IconButton
            size="small"
            aria-label={t`Copy a ${{ library: MONOMOD }} hook`}
            onClick={() => copyText(hookSnippet(hit), t`Hook copied`)}
          >
            <Copy size={14} />
          </IconButton>
        </Tooltip>
      )}
    </Box>
  )
}

// The line declaring method in decompiled source: its name before a parameter list, on a line that is not a call.
function declarationLine(source: string, type: string, method: string): number {
  const name = CONSTRUCTOR_NAME.test(method) ? (type.split(TYPE_PARTS).at(-1) ?? '') : method
  return (
    source
      .split('\n')
      // IL writes a space before the parameter list.
      .findIndex(
        (line) =>
          (line.includes(` ${name}(`) || line.includes(` ${name} (`)) &&
          !line.trimEnd().endsWith(';'),
      )
  )
}

function Source({
  game,
  dir,
  assembly,
  type,
  method,
  il,
}: {
  game: string
  dir: string
  assembly: string
  type: string
  method: string
  il: boolean
}) {
  const { t } = useLingui()
  const [has, setHas] = useState<boolean | null>(null)
  const [source, setSource] = useState('')
  const [installing, install] = usePending()
  useEffect(() => {
    HasDecompiler()
      .then(setHas)
      .catch(() => setHas(false))
  }, [])
  useEffect(() => {
    if (!has || type === '') {
      return
    }
    setSource('')
    Decompile(game, dir, assembly, type, il)
      .then(setSource)
      .catch(reportError(t`Could not decompile ${{ type }}`))
  }, [game, dir, assembly, type, il, has, t])
  if (has === false) {
    return (
      <EmptyState
        icon={<Code size={40} aria-hidden={true} />}
        title={t`Install ${{ loader: ILSPY }} to read the game's code`}
        action={
          <Button
            variant="contained"
            disabled={installing}
            onClick={() =>
              install(
                async () => {
                  await InstallDecompiler()
                  setHas(await HasDecompiler())
                },
                { errorTitle: t`Could not install ${{ loader: ILSPY }}` },
              )
            }
          >
            {installing ? t`Installing…` : t`Install ${{ loader: ILSPY }}`}
          </Button>
        }
      >
        {t`Mortar installs its command line as a .NET tool and decompiles the type you pick.`}
      </EmptyState>
    )
  }
  if (type === '') {
    return (
      <EmptyState icon={<FileSearch size={40} aria-hidden={true} />} title={t`Pick a method`}>
        {t`Search the game's types and methods, then pick one to read its type's code.`}
      </EmptyState>
    )
  }
  if (source === '') {
    return <LoadingRow>{t`Decompiling ${{ type }}…`}</LoadingRow>
  }
  return <CodeView source={source} il={il} line={declarationLine(source, type, method)} />
}

// Lists the methods, in every assembly the view lists, that use the picked member; picking one opens it.
function UsagesPanel({
  game,
  dir,
  target,
  onOpen,
  children,
}: {
  game: string
  dir: string
  target: string
  onOpen: (u: Usage) => void
  children: ReactNode
}) {
  const { t } = useLingui()
  const [usages, setUsages] = useState<Usage[] | null>(null)
  const [finding, find] = usePending()
  return (
    <Box sx={{ px: 2, pt: 1.5, display: 'flex', flexDirection: 'column', gap: 1 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
        <Button
          size="small"
          variant="outlined"
          disabled={target === '' || finding}
          onClick={() =>
            find(async () => setUsages(await Usages(game, dir, target)), {
              errorTitle: t`Could not find usages`,
            })
          }
        >
          {finding ? t`Finding usages…` : t`Find usages`}
        </Button>
        <Box sx={{ flex: 1 }} />
        {children}
      </Box>
      {usages === null ? null : (
        <Box sx={{ maxHeight: 160, overflowY: 'auto' }}>
          {usages.length === 0 ? (
            <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
              {t`Nothing in these assemblies uses it.`}
            </Typography>
          ) : null}
          {usages.map((u) => (
            <Box
              key={`${u.assembly}|${u.caller}`}
              component="button"
              type="button"
              onClick={() => onOpen(u)}
              sx={{ all: 'unset', display: 'block', cursor: 'pointer', py: 0.25 }}
            >
              <Typography noWrap={true} sx={{ fontFamily: MONO, fontSize: 'small.fontSize' }}>
                {u.caller}
              </Typography>
              <Typography
                noWrap={true}
                sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}
              >
                {u.label}
              </Typography>
            </Box>
          ))}
        </Box>
      )}
    </Box>
  )
}

/** Search the game's code, read a type decompiled, and copy a hook for one of its methods. */
export function GameCodeDialog({
  game,
  dir,
  onClose,
}: {
  game: string
  dir: string
  onClose: () => void
}) {
  const { t } = useLingui()
  const [query, setQuery] = useState('')
  const [picked, setPicked] = useState<Member | null>(null)
  const [assembly, setAssembly] = useState('')
  const [il, setIl] = useState(false)
  const assemblies =
    useLoaded(() => Assemblies(game, dir), [game, dir], null, reportUnexpected).data ?? []
  const { hits, error } = useHits(game, dir, assembly, query)
  return (
    <AppDialog
      open={true}
      onClose={onClose}
      title={t`Game code`}
      icon={
        <ToneIcon tone="neutral">
          <Code size={22} />
        </ToneIcon>
      }
      size="xl"
      height="80vh"
      flush={true}
    >
      <Box sx={{ display: 'grid', gridTemplateColumns: '380px minmax(0, 1fr)', height: '100%' }}>
        <Box
          sx={{
            display: 'flex',
            flexDirection: 'column',
            gap: space.gap,
            p: 2,
            minHeight: 0,
            borderRight: '1px solid var(--mortar-hairline-muted)',
          }}
        >
          <SelectField
            label={t`Assembly`}
            displayEmpty={true}
            value={assembly}
            onChange={(next) => {
              setAssembly(next)
              setPicked(null)
            }}
          >
            {assemblies.map((a, i) => (
              <MenuItem key={a.path} value={i === 0 && a.group === 'game' ? '' : a.path}>
                {a.label}
              </MenuItem>
            ))}
          </SelectField>
          <SearchField
            value={query}
            onChange={setQuery}
            label={t`Search the game's code`}
            placeholder={t`Type or method, such as player update`}
          />
          {error ? (
            <Typography role="alert" sx={{ fontSize: 'small.fontSize', color: 'error.light' }}>
              {error}
            </Typography>
          ) : null}
          <Box sx={{ flex: 1, minHeight: 0, overflowY: 'auto' }}>
            {hits.map((h) => (
              <HitRow
                key={`${h.type}::${h.method}(${(h.params ?? []).join(',')})`}
                hit={h}
                selected={picked === h}
                onPick={() => setPicked(h)}
              />
            ))}
          </Box>
        </Box>
        <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: 0, minWidth: 0 }}>
          <UsagesPanel
            key={picked ? `${picked.type}::${picked.method}` : ''}
            game={game}
            dir={dir}
            target={picked ? `${picked.type}::${picked.method}` : ''}
            onOpen={(u) => {
              const [type = '', method = ''] = u.caller.split('::')
              setAssembly(u.assembly === assemblies[0]?.path ? '' : u.assembly)
              setPicked({ type, kind: 'method', method, static: false, void: true, params: [] })
            }}
          >
            <ToggleButtonGroup
              exclusive={true}
              size="small"
              value={il ? 'il' : 'cs'}
              onChange={(_, next: string | null) => next && setIl(next === 'il')}
              aria-label={t`Code view`}
            >
              <ToggleButton value="cs">C#</ToggleButton>
              <ToggleButton value="il">IL</ToggleButton>
            </ToggleButtonGroup>
          </UsagesPanel>
          <Source
            game={game}
            dir={dir}
            assembly={assembly}
            type={picked?.type ?? ''}
            method={picked?.method ?? ''}
            il={il}
          />
        </Box>
      </Box>
    </AppDialog>
  )
}
