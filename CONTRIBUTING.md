# Contributing

This is the workflow for maintainers and agents; the rules the code keeps are in [`AGENTS.md`](AGENTS.md), and every command is in [`HUMANS.md`](HUMANS.md).

## Sending a change

Report a bug or suggest a feature through the [issue forms](https://github.com/Rethunk-Tech/mortar/issues/new/choose); report a security problem privately as [`SECURITY.md`](SECURITY.md) says. Changes arrive as pull requests against `main` from a fork or branch: fill in the [pull request template](.github/pull_request_template.md), and CI runs the same gate on every pull request. Translations go through [Hosted Weblate](https://hosted.weblate.org/projects/mortar/) instead.

## Before review

Run `bun run gate` ([`HUMANS.md`](HUMANS.md#gate)); the lefthook pre-push hook runs `gate`, which covers it. Nothing is merged on a red gate, and no hook is bypassed.

## Commits

- Conventional Commits: `type(scope): subject`, one logical change each.
- Stage explicit paths only.
- The body, when there is one, says why; the diff already says what.

## Pending work

Decided work that is not built yet is specified in [`docs/design.md`](docs/design.md). When an item lands, delete it there in the same change.
