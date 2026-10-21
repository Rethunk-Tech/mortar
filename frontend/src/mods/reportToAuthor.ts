import type { I18n } from '@lingui/core'
import {
  type Entry,
  Level,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import type { Source } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { format } from '../console/filter.ts'
import { nexusModUrl } from './nexusUrl.ts'

const MAX_ERROR_LINES = 20
const MAX_GITHUB_BODY = 6000
// The longest address Mortar's opener accepts (internal/opener maxURL).
const MAX_URL = 2048

interface ModReportFields {
  modName: string
  modVersion: string
  gameName: string
  gameVersion: string
  smapiVersion: string
  mortarVersion: string
  errorLines: readonly string[]
  logShareURL: string
  source: Source | null | undefined
  nexusDomain: string
  githubRepo: string
  nexusModId: number
  // pageUrl is where the mod's own site takes reports when it has no GitHub repository or Nexus bug tracker.
  pageUrl: string
  issueTitle: string
}

interface ModReportResult {
  text: string
  url: string
  nexus: boolean
  github: boolean
  page: boolean
  // The issue link left part of the text out; the whole text is on the clipboard.
  cut: boolean
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

const GITHUB_REPO = /^[A-Za-z0-9._-]+\/[A-Za-z0-9._-]+$/

// Mirrors manifest.ValidGitHubRepo: "." and "..", which the pattern admits, would move the URL's path.
function validGithubRepo(repo: string): boolean {
  const [owner = '', name = ''] = repo.split('/')
  return (
    GITHUB_REPO.test(repo) && owner.replaceAll('.', '') !== '' && name.replaceAll('.', '') !== ''
  )
}

function githubIssueURL(repo: string, title: string, body: string): { url: string; cut: boolean } {
  const trimmed = repo.trim()
  if (!validGithubRepo(trimmed)) {
    return { url: '', cut: false }
  }
  const base = `https://github.com/${trimmed.split('/').map(encodeURIComponent).join('/')}/issues/new?`
  const at = (n: number) => base + new URLSearchParams({ title, body: body.slice(0, n) }).toString()
  // Encoding grows each character by up to nine, so the longest body that fits is found by halving.
  let lo = 0
  let hi = Math.min(body.length, MAX_GITHUB_BODY)
  while (lo < hi) {
    const mid = Math.ceil((lo + hi) / 2)
    if (at(mid).length <= MAX_URL) {
      lo = mid
    } else {
      hi = mid - 1
    }
  }
  return { url: at(lo).length <= MAX_URL ? at(lo) : '', cut: lo < body.length }
}

function nexusBugsURL(domain: string, modId: number): string {
  if (domain === '' || modId <= 0) {
    return ''
  }
  return nexusModUrl(modId, domain, 'bugs')
}

function buildModReportText(_i18n: I18n, fields: ModReportFields): string {
  const lines: string[] = [`Mod: ${fields.modName} ${fields.modVersion.trim()}`.trimEnd()]
  if (fields.gameVersion !== '') {
    lines.push(`Game: ${fields.gameName} ${fields.gameVersion}`)
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
  const { issueTitle, modName, source, githubRepo, nexusModId, nexusDomain, pageUrl } = fields
  const title = issueTitle === '' ? `Error in ${modName}` : issueTitle
  let repo = githubRepo
  if (repo === '' && source?.kind === 'github') {
    repo = source.repo ?? ''
  }
  let modId = nexusModId
  if (modId <= 0 && source?.kind === 'nexus') {
    modId = source.modId ?? 0
  }
  if (repo.trim() !== '') {
    const { url, cut } = githubIssueURL(repo, title, text)
    return { text, url, nexus: false, github: true, page: false, cut }
  }
  if (modId > 0 && nexusDomain !== '') {
    return {
      text,
      url: nexusBugsURL(nexusDomain, modId),
      nexus: true,
      github: false,
      page: false,
      cut: false,
    }
  }
  return { text, url: pageUrl, nexus: false, github: false, page: pageUrl !== '', cut: false }
}

function buildModReport(i18n: I18n, fields: ModReportFields): ModReportResult {
  const text = buildModReportText(i18n, fields)
  return buildAuthorReportUrl(text, fields)
}

export type { ModReportFields }
export { buildModReport, buildModReportText, githubIssueURL, modErrorLines, truncateGithubBody }
