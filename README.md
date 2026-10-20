# bnbloc-local-skills

**A member's own skills, in their agents' sandbox and their library, for [bnbloc](https://github.com/luckylhb90/bnbloc).**

An installable plugin sidecar (`bnbloc.plugin.package/v1`, capabilities
`skills-provider` and `sandbox-mounts`). The member chooses, skill by skill
on the Components page (the component's card: Open), which skill folders on
their computer reach their agents' sandbox and which are added to their
library. Off, and empty, until they turn it on. macOS desktops only for now:
the sandbox's guest backend is macOS-only.

Adding a skill to the library goes through a publish step in the panel: the
member sees the files and sets the library entry (name, display name,
description, tags), and may first have it reviewed with AI (bnbloc's
`review-skill` assist reads SKILL.md and the file list, says what to fix,
such as a secret or a personal path left in, and proposes the entry, applied
field by field). Publishing stays the member's decision.

It speaks two contracts: `bnbloc.skills.provider/v1` (the skills panel bnbloc
renders, over loopback with a per-launch bearer) and `bnbloc.sandbox.mounts/v1` (the
descriptor the bridge reads before a sandboxed turn; the types are bnbloc's
public `component` package). Design:
[local skills](https://github.com/luckylhb90/bnbloc/blob/main/docs/architecture/local-skills.md).

## Settings

What the component works to is its shipped policy
(`localskills/policy.default.json`: the agent folders it knows, what is never
copied, what marks a credential, its limits). The settings `package.json`
declares add to it, and are set on the component's card on the Components
page (Settings):

| Setting | Type | Set by | Does |
|---|---|---|---|
| `extraSkillDirs` | lines | each member | extra agent skill folders under the member's home (`~/.mytool/skills`); a skill linked from one is also placed there in the sandbox. A line that is not a folder under the home is left out, and the skills panel says so. |
| `excludes` | lines | each member | name patterns never copied into the sandbox or the library (`*.log`) |
| `secretNames` | lines | the organization | file name patterns that hold credentials (`*.secret`): a skill holding one is never shared |
| `secretPatterns` | lines | the organization | regular expressions (Go syntax) of text that marks a credential; one that does not compile stops the component with an error naming it, never ignored |
| `maxSkills` | number | the organization | how many skills one sandbox takes at most (empty: the policy's limit) |

Each list is added after the shipped policy's, each entry once
(`localskills.LoadPolicy`, one merge: `Policy.Merge`); nothing a setting says
can remove a rule the component ships. The values reach the component through
the file bnbloc names in `BNBLOC_COMPONENT_SETTINGS_FILE` (an owner-only JSON
object by key: a `lines` setting is an array of strings, a number its decimal
text, an unset setting absent). It is read again on every call, so a changed
value applies without a restart. Without the variable the shipped policy
applies alone; a named file that cannot be read or parsed is an error.

## Install into bnbloc

Download the binaries and `package.json` from the
[releases](https://github.com/luckylhb90/bnbloc-local-skills/releases) (or build them,
below), then as a bnbloc super-admin:

```sh
bnbloc components publish-package --gateway https://<gateway> --package package.json \
  --binary darwin-arm64=dist/bnbloc-local-skills-darwin-arm64 --binary darwin-amd64=dist/bnbloc-local-skills-darwin-amd64 \
  --rollout optin
```

Members who opt in (or everyone, with `--rollout auto`) get it in their
desktop app.

## Build and test

```sh
go test ./...
go build -o bnbloc-local-skills .
```

It builds against bnbloc's public packages (`component`, `core`,
`core/pkgmeta`), pinned in `go.mod`. A tag `vX.Y.Z` builds every platform
binary in CI and attaches them to the release; the tag's version is the
`version` in `package.json`.

## License

[Apache License 2.0](LICENSE).
