import type { I18n } from '@lingui/core'
import {
  type Entry,
  Level,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import type { Source } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { format } from '../console/filter.ts'

const MAX_ERROR_LINES = 20
const MAX_GITHUB_BODY = 6000
const STARDEW_NEXUS_DOMAIN = 'stardewvalley'

interface ModReportFields {
  modName: string
  modVersion: string
  gameVersion: string
  smapiVersion: string
  mortarVersion: string
  errorLines: readonly string[]
  logShareURL: string
  source: Source | null | undefined
  nexusDomain: string
  githubRepo: string
  nexusModId: number
  issueTitle: string
}

interface ModReportResult {
  text: string
  url: string
  nexus: boolean
  github: boolean
}

function modErrorLines(entries: readonly Entry[], modName: string): string[] {
  if (modName === '') {
    return []
  }
  const out: string[] = []
  for (let i = 0; i < entries.length && out.length < MAX_ERROR_LINES; i++) {
    const header = entries[i]
    const isModError =
      header &&
      !header.cont &&
      (header.level === Level.Error || header.level === Level.Alert) &&
      header.mod === modName
    if (isModError) {
      for (let j = i; j < entries.length && (j === i || entries[j]?.cont); j++) {
        const row = entries[j]
        if (!row) {
          break
        }
        out.push(format(row))
        if (out.length >= MAX_ERROR_LINES) {
          return out
        }
      }
    }
  }
  return out
}

function truncateGithubBody(body: string): string {
  if (body.length <= MAX_GITHUB_BODY) {
    return body
  }
  return body.slice(0, MAX_GITHUB_BODY)
}

function githubIssueURL(repo: string, title: string, body: string): string {
  const trimmed = repo.trim()
  if (trimmed === '') {
    return ''
  }
  const q = new URLSearchParams({
    title,
    body: truncateGithubBody(body),
  })
  return `https://github.com/${trimmed}/issues/new?${q.toString()}`
}

function nexusBugsURL(domain: string, modId: number): string {
  if (domain === '' || modId <= 0) {
    return ''
  }
  return `https://www.nexusmods.com/${domain}/mods/${modId}?tab=bugs`
}

function buildModReportText(_i18n: I18n, fields: ModReportFields): string {
  const lines: string[] = [`Mod: ${fields.modName} ${fields.modVersion.trim()}`.trimEnd()]
  if (fields.gameVersion !== '') {
    lines.push(`Game: Stardew Valley ${fields.gameVersion}`)
  }
  if (fields.smapiVersion !== '') {
    lines.push(`SMAPI: ${fields.smapiVersion}`)
  }
  if (fields.mortarVersion !== '') {
    lines.push(`Mortar: ${fields.mortarVersion}`)
  }
  lines.push('', 'Errors:')
  if (fields.errorLines.length === 0) {
    lines.push('(none captured)')
  } else {
    lines.push(...fields.errorLines)
  }
  if (fields.logShareURL !== '') {
    lines.push('', `Log: ${fields.logShareURL}`)
  }
  return `${lines.join('\n')}\n`
}

function buildAuthorReportUrl(text: string, fields: ModReportFields): ModReportResult {
  const { issueTitle, modName, source, githubRepo, nexusModId, nexusDomain } = fields
  const title = issueTitle === '' ? `Error in ${modName}` : issueTitle
  let repo = githubRepo
  if (repo === '' && source?.kind === 'github') {
    repo = source.repo ?? ''
  }
  let modId = nexusModId
  if (modId <= 0 && source?.kind === 'nexus') {
    modId = source.modId ?? 0
  }
  const domain = nexusDomain === '' ? STARDEW_NEXUS_DOMAIN : nexusDomain
  if (repo.trim() !== '') {
    return {
      text,
      url: githubIssueURL(repo, title, text),
      nexus: false,
      github: true,
    }
  }
  if (modId > 0 && domain !== '') {
    return {
      text,
      url: nexusBugsURL(domain, modId),
      nexus: true,
      github: false,
    }
  }
  return { text, url: '', nexus: false, github: false }
}

function buildModReport(i18n: I18n, fields: ModReportFields): ModReportResult {
  const text = buildModReportText(i18n, fields)
  return buildAuthorReportUrl(text, fields)
}

export type { ModReportFields, ModReportResult }
export {
  buildAuthorReportUrl,
  buildModReport,
  buildModReportText,
  githubIssueURL,
  MAX_ERROR_LINES,
  MAX_GITHUB_BODY,
  modErrorLines,
  nexusBugsURL,
  STARDEW_NEXUS_DOMAIN,
  truncateGithubBody,
}
