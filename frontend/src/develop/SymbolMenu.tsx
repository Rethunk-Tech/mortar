import { useLingui } from '@lingui/react/macro'
import { Menu } from '@mui/material'
import { useLoaded } from '@rethunk/react-load-hooks'
import { Copy, CornerDownRight, Search } from 'lucide-react'
import {
  DecompileMapped,
  GameCode,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import type { Member } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/dotnet/models.ts'
import { MONOMOD } from '../brand/productNames.ts'
import { copyText } from '../share/copyText.ts'
import { MenuAction, MenuNote } from '../shell/MenuAction.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { declares } from './declaration.ts'
import { hookSnippet, memberKey } from './hook.ts'
import { ilOwner, type SymbolClick } from './symbolAt.ts'

// Members a right-click menu lists, at most.
const MENU_MEMBERS = 6

// Where a right-click landed: the assembly and the type open in the view.
interface Scope {
  game: string
  dir: string
  assembly: string
  type: string
}

// The type that owns a right-clicked name: the open type for its own declaration, else the target IL names on that
// line (read from the IL view, or through the line map from C#), else the open type for a name not after a dot.
// '' when none of those settles it.
async function ownerOf(scope: Scope, at: SymbolClick): Promise<string> {
  if (scope.type !== '' && declares(at.text, at.name)) {
    return scope.type
  }
  if (at.il) {
    return ilOwner(at.text, at.name)
  }
  if (scope.type === '') {
    return ''
  }
  const mapped = await DecompileMapped(scope.game, scope.dir, scope.assembly, scope.type)
  const il = mapped.il.split('\n')
  for (const [i, cs] of (mapped.links ?? []).entries()) {
    const owner = cs === at.line ? ilOwner(il[i] ?? '', at.name) : ''
    if (owner !== '') {
      return owner
    }
  }
  const before = at.text.slice(0, at.text.search(new RegExp(`\\b${at.name}\\b`))).trimEnd()
  return before.endsWith('.') ? '' : scope.type
}

/**
 * What a right-clicked name can do: go to its definition from a use of it, find who uses the member it names, and
 * copy a hook for a method, whose signature the search gives. Only when its owner cannot be told are all the
 * assembly's members by that name offered.
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
  const name = at?.name ?? ''
  const { game, dir, assembly, type } = scope
  const { data: found } = useLoaded<{ owner: string; members: Member[] } | null>(
    at === null
      ? null
      : async () => {
          const owner = await ownerOf({ game, dir, assembly, type }, at)
          const hits =
            (await GameCode(game, dir, assembly, owner ? `${owner}::${at.name}` : at.name)) ?? []
          return {
            owner,
            members: hits.filter((m) => m.method === at.name && (owner === '' || m.type === owner)),
          }
        },
    [game, dir, assembly, type, at],
    null,
    reportUnexpected,
  )
  const members = found?.members.slice(0, MENU_MEMBERS) ?? []
  const onItsDefinition = at !== null && declares(at.text, at.name)
  // A member of another assembly (Unity's, say) is not in this one's search, but its usages can still be found.
  const outside =
    found && found.owner !== '' && members.length === 0 ? `${found.owner}::${name}` : ''
  return (
    <Menu
      open={at !== null}
      onClose={onClose}
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
        const label = `${m.type.split('.').at(-1) ?? m.type}::${m.method}`
        // A code signature, not words, so it stays out of the translated message whole.
        const signature = `${label}(${(m.params ?? []).join(', ')})`
        return [
          onItsDefinition ? null : (
            <MenuAction
              key={`d${memberKey(m)}`}
              icon={<CornerDownRight size={14} />}
              label={t`Go to the definition of ${{ label }}`}
              onClick={() => onDefinition(m)}
            />
          ),
          <MenuAction
            key={`u${memberKey(m)}`}
            icon={<Search size={14} />}
            label={t`Find usages of ${{ label }}`}
            onClick={() => onUsages(`${m.type}::${m.method}`)}
          />,
          m.kind === 'field' ? null : (
            <MenuAction
              key={`h${memberKey(m)}`}
              icon={<Copy size={14} />}
              label={t`Copy a ${{ library: MONOMOD }} hook for ${{ signature }}`}
              onClick={() => {
                copyText(hookSnippet(m), t`Hook copied`)
                onClose()
              }}
            />
          ),
        ]
      })}
    </Menu>
  )
}
