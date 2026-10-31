// The same address forms as devsvc's ParseRepo: owner/name, or a github.com address over https or git@.
const GITHUB_REPO =
  /^(?:https:\/\/github\.com\/|git@github\.com:)?[A-Za-z0-9][A-Za-z0-9-]{0,38}\/[A-Za-z0-9._-]{1,100}?(?:\.git)?\/?$/

const TRAILING_SLASHES = /\/+$/
const DOT_GIT = /\.git$/
const NAME_START = /^.*[/:]/
const TRAILING_SEPARATORS = /[\\/]+$/

/** Where Clone puts the project of repo (an address Clone accepts) below parent. */
export function cloneFolder(parent: string, repo: string): string {
  const name = repo
    .trim()
    .replace(TRAILING_SLASHES, '')
    .replace(DOT_GIT, '')
    .replace(NAME_START, '')
  return parent.replace(TRAILING_SEPARATORS, '') + (parent.includes('\\') ? '\\' : '/') + name
}

/** Whether text names a GitHub repository Clone accepts. */
export function isRepoAddress(text: string): boolean {
  return GITHUB_REPO.test(text.trim())
}
