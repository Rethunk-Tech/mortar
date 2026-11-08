import { useLingui } from '@lingui/react/macro'
import { Menu } from '@mui/material'
import { useLoaded } from '@rethunk/react-load-hooks'
import { ChevronRight, Copy, CornerDownRight, Search } from 'lucide-react'
import { useState } from 'react'
import {
  DecompileMapped,
  GameCode,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import type { Member } from '../../bindings/github.com/Rethunk-Tech/go-dotnet/models.ts'
import { HARMONY, HOOKGEN, MONOMOD } from '../brand/productNames.ts'
import { copyText } from '../share/copyText.ts'
import { escapeRegExp } from '../shell/escapeRegExp.ts'
import { MenuAction, MenuNote } from '../shell/MenuAction.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { declares } from './declaration.ts'
import { harmonyPatchSnippet, hookSnippet, memberKey, monoModHookSnippet } from './hook.ts'
import { csArity, ilArity, ilOwner, type SymbolClick } from './symbolAt.ts'

// Members a right-click menu lists, at most.
const MENU_MEMBERS = 6

interface Found {
  owner: string
  members: Member[]
}

// What each right-click resolved to, so the same name on the same line opens its menu at once the next time. The game's
// code does not change while Mortar runs.
const lookups = new Map<string, Found>()
const LOOKUPS_MAX = 500

// Where a right-click landed: the assembly and the type open in the view.
interface Scope {
  game: string
  dir: string
  assembly: string
  type: string
}

// Who a right-clicked name is: the type that owns it and, for a method, how many arguments the call passes or the
// declaration lists (-1 when unknown), which tells overloads apart. The owner is the open type for its own declaration,
// else the target IL names on that line (read from the IL view, or through the line map from C#), else the open type
// for a name not after a dot; '' when none of those settles it.
async function ownerOf(scope: Scope, at: SymbolClick): Promise<{ owner: string; arity: number }> {
  if (scope.type !== '' && declares(at.text, at.name)) {
    return { owner: scope.type, arity: at.il ? -1 : csArity(at.text, at.name) }
  }
  if (at.il) {
    return { owner: ilOwner(at.text, at.name), arity: ilArity(at.text, at.name) }
  }
  if (scope.type === '') {
    return { owner: '', arity: -1 }
  }
  const mapped = await DecompileMapped(scope.game, scope.dir, scope.assembly, scope.type)
  const il = mapped.il.split('\n')
  for (const [i, cs] of (mapped.links ?? []).entries()) {
    const line = cs === at.line ? (il[i] ?? '') : ''
    const owner = line === '' ? '' : ilOwner(line, at.name)
    if (owner !== '') {
      return { owner, arity: ilArity(line, at.name) }
    }
  }
  const before = at.text
    .slice(0, at.text.search(new RegExp(`\\b${escapeRegExp(at.name)}\\b`)))
    .trimEnd()
  return { owner: before.endsWith('.') ? '' : scope.type, arity: -1 }
}

// What a right-click resolves to: a name already looked up on that line comes from the cache at once.
function useLookup(scope: Scope, at: SymbolClick | null): Found | null {
  const { game, dir, assembly, type } = scope
  const lookupKey = at ? [game, dir, assembly, type, at.il, at.line, at.name].join('\n') : ''
  const cached = lookups.get(lookupKey)
  const { data: loaded } = useLoaded<Found | null>(
    at === null || cached
      ? null
      : async () => {
          const { owner, arity } = await ownerOf({ game, dir, assembly, type }, at)
          const hits =
            (await GameCode(game, dir, assembly, owner ? `${owner}::${at.name}` : at.name)) ?? []
          const resolved = {
            owner,
            members: overload(
              hits.filter((m) => m.method === at.name && (owner === '' || m.type === owner)),
              arity,
            ),
          }
          if (lookups.size >= LOOKUPS_MAX) {
            lookups.clear()
          }
          lookups.set(lookupKey, resolved)
          return resolved
        },
    [lookupKey],
    null,
    reportUnexpected,
  )
  return cached ?? loaded
}

// The overloads that take arity arguments, when that leaves any; all of them when the count is unknown.
function overload(members: Member[], arity: number): Member[] {
  const taking = members.filter((m) => m.kind === 'method' && (m.params ?? []).length === arity)
  return arity < 0 || taking.length === 0 ? members : taking
}

/**
 * What a right-clicked name can do: go to its definition from a use of it, find who uses the member it names, and on a
 * method's own declaration copy a HookGen hook, a MonoMod Hook or a Harmony patch for it, built from the signature the
 * search gives. Only when its owner cannot be told are all the assembly's members by that name offered.
 */
export function SymbolMenu({
  scope,
  at,
  onClose,
  onUsages,
  onDefinition,
}: {
  scope: Scope
  at: SymbolClick | null
  onClose: () => void
  onUsages: (target: string) => void
  onDefinition: (m: Member) => void
}) {
  const { t } = useLingui()
  // The Copy hook row whose submenu is open, and its member.
  const [hooks, setHooks] = useState<{ row: HTMLElement; member: Member } | null>(null)
  const close = () => {
    setHooks(null)
    onClose()
  }
  const name = at?.name ?? ''
  const found = useLookup(scope, at)
  const members = found?.members.slice(0, MENU_MEMBERS) ?? []
  const onItsDefinition = at !== null && declares(at.text, at.name)
  // A member of another assembly (Unity's, say) is not in this one's search, but its usages can still be found.
  const outside =
    found && found.owner !== '' && members.length === 0 ? `${found.owner}::${name}` : ''
  return (
    <Menu
      open={at !== null}
      onClose={close}
      anchorReference="anchorPosition"
      anchorPosition={at ? { top: at.y, left: at.x } : undefined}
      // Above the full-window dialog it opens from.
      sx={{ zIndex: (theme) => theme.zIndex.modal + 1 }}
    >
      {found === null ? <MenuNote>{t`Looking up ${{ name }}…`}</MenuNote> : null}
      {found !== null && members.length === 0 && outside === '' ? (
        <MenuNote>{t`No member named ${{ name }} in this assembly`}</MenuNote>
      ) : null}
      {outside === '' ? null : (
        <MenuAction
          icon={<Search size={14} />}
          label={t`Find usages of ${{ label: outside }}`}
          onClick={() => onUsages(outside)}
        />
      )}
      {members.flatMap((m) => {
        const key = memberKey(m)
        return [
          // Only when the name could be more than one member does each group say whose it is.
          members.length > 1 ? (
            <MenuNote key={`n${key}`}>
              {`${m.type.split('.').at(-1) ?? m.type}::${m.method}${m.kind === 'field' ? '' : `(${(m.types ?? []).join(', ')})`}`}
            </MenuNote>
          ) : null,
          onItsDefinition ? null : (
            <MenuAction
              key={`d${key}`}
              icon={<CornerDownRight size={14} />}
              label={t`Go to definition`}
              onClick={() => onDefinition(m)}
            />
          ),
          <MenuAction
            key={`u${key}`}
            icon={<Search size={14} />}
            label={t`Find usages`}
            onClick={() => onUsages(`${m.type}::${m.method}`)}
          />,
          onItsDefinition && m.kind !== 'field' ? (
            <MenuAction
              key={`h${key}`}
              icon={<Copy size={14} />}
              label={t`Copy hook`}
              expanded={hooks?.member === m}
              end={<ChevronRight size={14} />}
              onClick={(e) => setHooks({ row: e.currentTarget, member: m })}
            />
          ) : null,
        ]
      })}
      <Menu
        open={hooks !== null}
        anchorEl={hooks?.row}
        onClose={() => setHooks(null)}
        anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'left' }}
        sx={{ zIndex: (theme) => theme.zIndex.modal + 2 }}
      >
        {hooks
          ? [
              { id: 'on', text: t`${{ app: HOOKGEN }} On. hook`, make: hookSnippet },
              { id: 'mm', text: t`${{ app: MONOMOD }} Hook`, make: monoModHookSnippet },
              { id: 'hp', text: t`${{ app: HARMONY }} patch`, make: harmonyPatchSnippet },
            ].map((h) => (
              <MenuAction
                key={h.id}
                icon={<Copy size={14} />}
                label={h.text}
                onClick={() => {
                  copyText(h.make(hooks.member), t`Hook copied`)
                  close()
                }}
              />
            ))
          : null}
      </Menu>
    </Menu>
  )
}
