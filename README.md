# bnbloc-local-skills

**A member's own skills, in their agents' sandbox and their library, for [bnbloc](https://github.com/luckylhb90/bnbloc).**

An installable plugin sidecar (`bnbloc.plugin.package/v1`, capabilities
`skills-provider` and `sandbox-mounts`). The member chooses, skill by skill
in Settings → Skills, which skill folders on their computer reach their
agents' sandbox and which are added to their library. Off, and empty, until
they turn it on.

It speaks two contracts: `bnbloc.skills.provider/v1` (what Settings renders,
over loopback with a per-launch bearer) and `bnbloc.sandbox.mounts/v1` (the
descriptor the bridge reads before a sandboxed turn; the types are bnbloc's
public `component` package). Design:
[local skills](https://github.com/luckylhb90/bnbloc/blob/main/docs/architecture/local-skills.md).

## Install into bnbloc

Download the binaries and `package.json` from the
[releases](https://github.com/luckylhb90/bnbloc-local-skills/releases) (or build them,
below), then as a bnbloc super-admin:

```sh
bnbloc components publish-package --gateway https://<gateway> --package package.json \
  --binary darwin-arm64=dist/bnbloc-local-skills-darwin-arm64 --binary darwin-amd64=dist/bnbloc-local-skills-darwin-amd64 \
  --binary windows-amd64=dist/bnbloc-local-skills-windows-amd64.exe --rollout optin
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
