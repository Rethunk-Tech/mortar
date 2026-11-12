import type { Input } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/modreport/models.ts'
import type { Source } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

export interface ReportParts {
  game: string
  modName: string
  modVersion: string
  // The name the mod's own log lines carry, whose errors the report lists.
  modLogName: string
  source: Source | null | undefined
  loader: { gameVersion: string; version: string } | null
  loaderName: string
  mortarVersion: string
  logShareURL: string
  issueTitle: string
  // Where the mod's site takes reports (see ReportTarget); the repo is used when the mod itself is not from GitHub.
  target: { repo: string; url: string }
}

// The Go side words the report and picks the link; the window supplies only what it alone knows.
export function reportInput(p: ReportParts): Input {
  return {
    ModName: p.modName,
    ModVersion: p.modVersion,
    Game: p.game,
    GameVersion: p.loader?.gameVersion ?? '',
    SMAPIVersion: p.loader?.version ?? '',
    MortarVersion: p.mortarVersion,
    LoaderName: p.loaderName,
    ErrorLines: null,
    Entries: null,
    LogName: p.modLogName,
    LogShareURL: p.logShareURL,
    Source: p.source ?? { kind: '', name: '' },
    NexusDomain: '',
    GitHubRepo: p.source?.kind === 'github' ? '' : p.target.repo,
    NexusModID: 0,
    IssueTitle: p.issueTitle,
    PageURL: p.target.url,
  }
}
