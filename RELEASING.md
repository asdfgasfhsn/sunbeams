# Releasing

sunbeams uses [Semantic Versioning](https://semver.org/) (`vMAJOR.MINOR.PATCH`) and releases via [goreleaser](https://goreleaser.com/). Tags drive everything — push a `v*` tag and the GitHub Actions workflow builds, signs, and publishes the release.

## Version policy

| Bump | When |
|---|---|
| **MAJOR** | Breaking change to the CLI surface (flags, subcommand names, exit codes), the TOML config schema, or kernel-arg invariants that would require users to re-run `sunbeams install`. |
| **MINOR** | New subcommand, new flag, new config field — additive and backward compatible. |
| **PATCH** | Bug fix, doc update, dependency bump, refactor with no observable behaviour change. |

While on `v0.x`, the contract is weaker: the CLI surface may shift between minor bumps as the tool gets validated on real Bazzite hardware. Promote to `v1.0.0` only after the generator, switcher, and installer have been exercised end-to-end on a live system.

## Commit style

Conventional Commits so goreleaser's changelog can group them:

- `feat:` / `feat(scope):` — user-visible feature → MINOR
- `fix:` / `fix(scope):` — bug fix → PATCH
- `refactor:` / `chore:` / `ci:` / `test:` / `build:` — internal, usually PATCH
- `docs:` — docs only, PATCH (or no bump at all)
- Add `!` after the type or a `BREAKING CHANGE:` footer to signal MAJOR

## Release steps

1. Confirm the tree is green:

   ```bash
   make check        # fmt + lint + tests + golden-file verification
   make snapshot     # verifies goreleaser config, produces dist/ locally
   ```

2. Decide the new version based on the commits since the last tag:

   ```bash
   git log --oneline $(git describe --tags --abbrev=0 2>/dev/null || echo HEAD)..HEAD
   ```

3. Tag and push:

   ```bash
   VERSION=v0.2.0
   git tag -a "$VERSION" -m "Release $VERSION"
   git push origin main
   git push origin "$VERSION"
   ```

4. The `release.yml` workflow triggers on the tag push and runs goreleaser, which:
   - Cross-compiles static Linux `amd64` + `arm64` binaries.
   - Packs each with `README.md`, `LICENSE`, `CONTRIBUTING.md` into `sunbeams_<version>_linux_<arch>.tar.gz`.
   - Publishes a GitHub Release with a generated changelog (commits with `docs:`, `test:`, `ci:`, `chore:` prefixes are excluded).

5. Verify the release page at `https://github.com/asdfgasfhsn/sunbeams/releases/tag/<VERSION>` has both archives and a `checksums.txt`.

## Undoing a release

If a tag was pushed in error **before** the workflow published a release, delete it locally and on origin:

```bash
git tag -d "$VERSION"
git push origin ":refs/tags/$VERSION"
```

If the release was already published, mark it pre-release or draft in the GitHub UI and cut a new patch tag with the fix — do not delete a published release, since users may already have pulled the binaries.

## Local snapshots

For testing changes to `.goreleaser.yml` without tagging:

```bash
make snapshot
ls dist/
```

`dist/` is gitignored. Snapshot artifacts use the form `sunbeams_0.0.0-SNAPSHOT-<short-sha>_linux_<arch>.tar.gz`.
