// The same address forms as devsvc's ParseRepo: owner/name, or a github.com address over https or git@.
const GITHUB_REPO =
  /^(?:https:\/\/github\.com\/|git@github\.com:)?[A-Za-z0-9][A-Za-z0-9-]{0,38}\/[A-Za-z0-9._-]{1,100}?(?:\.git)?\/?$/

/** Whether text names a GitHub repository Clone accepts. */
export function isRepoAddress(text: string): boolean {
  return GITHUB_REPO.test(text.trim())
}
