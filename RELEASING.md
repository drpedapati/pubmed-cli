# Release Workflow

This checklist is the production gate for `pubmed-cli`.

## 1. Code Quality Gate

Run all of the following before tagging:

```bash
go build ./...
go test ./...
go vet ./...
```

Optional but recommended:

```bash
go test -race ./...
```

## 2. Real-User CLI Smoke Gate

Use a freshly built local binary and run at least these commands:

```bash
./pubmed --help
./pubmed search "autism" --limit 2 --human
./pubmed fetch "38000001,38000002" --json
./pubmed fetch "38000001,38000002" --ris /tmp/fetch.ris
./pubmed cited-by 38000001 --limit 2 --human
./pubmed references 38000001 --limit 2 --human
./pubmed related 38000001 --limit 2 --human
./pubmed related 38000001 --limit 2 --ris /tmp/related.ris
./pubmed mesh depression --human
```

Negative-path checks (must fail cleanly, no panics):

```bash
./pubmed related 38000001 --limit -1
./pubmed search autism --sort newest
./pubmed search autism --year 2025-2020
./pubmed fetch abc123
./pubmed search autism --ris /tmp/search.ris
./pubmed mesh depression --ris /tmp/mesh.ris
```

## 3. Documentation Gate

Update and review:
- `README.md`
- `CHANGELOG.md`
- `CODE_REVIEW.md`
- `UX_TESTING.md`
- `docs/index.html`
- `docs/testing-guide.html`
- `TDD.md`

Requirements:
- Command list matches actual `--help` output.
- Examples are executable on current branch.
- Validation behavior is documented.
- No references to removed AI-only commands on `main`.

## 4. Commit + Tag

```bash
git add -A
git status
git commit -m "prepare release vX.Y.Z"
git push origin main

git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z
```

## 5. Build Release Artifacts

```bash
make release
# Expected: pubmed-darwin-arm64, pubmed-darwin-amd64, pubmed-linux-amd64
```

## 6. Create GitHub Release

```bash
gh release create vX.Y.Z \
  pubmed-darwin-arm64 \
  pubmed-darwin-amd64 \
  pubmed-linux-amd64 \
  --title "vX.Y.Z" \
  --notes "See CHANGELOG.md for release notes."
```

## 7. Homebrew Formula Update

Compute SHA256:

```bash
curl -sL https://github.com/henrybloomingdale/pubmed-cli/releases/download/vX.Y.Z/pubmed-darwin-arm64 | shasum -a 256
curl -sL https://github.com/henrybloomingdale/pubmed-cli/releases/download/vX.Y.Z/pubmed-darwin-amd64 | shasum -a 256
```

Update formula in `~/github/homebrew-tools/Formula/pubmed-cli.rb`, then commit and push in that repo.

## 8. Post-Release Verification

```bash
brew update
brew upgrade pubmed-cli
pubmed --help
pubmed search "autism" --limit 1 --json
```

If Homebrew is removed locally during development, verify with a downloaded release binary instead.
