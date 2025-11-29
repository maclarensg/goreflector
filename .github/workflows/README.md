# GitHub Actions Workflows

This directory contains automated CI/CD workflows for goreflector.

## Workflows

### 1. CI Workflow (`ci.yml`)

**Triggers:**
- Push to any branch except `main`
- Pull requests to `main` branch

**Jobs:**

1. **Test**
   - Runs all tests with race detection
   - Generates code coverage report
   - **Enforces 80% minimum coverage** (fails if below)
   - Uploads coverage to Codecov

2. **Build**
   - Builds the binary for verification
   - Ensures code compiles successfully

3. **Lint**
   - Runs golangci-lint with strict rules
   - Checks code style and quality

4. **Security**
   - Runs gosec security scanner
   - Uploads SARIF results for GitHub Security tab
   - Fails on security vulnerabilities

**Coverage Requirement:**
The CI enforces a minimum of **65% test coverage**. PRs with lower coverage will fail.

### 2. Release Workflow (`release.yml`)

**Triggers:**
- When a GitHub release is created (via GitHub UI or `gh release create`)

**Build Targets:**

**Linux:**
- `goreflector-linux-amd64.tar.gz` (Intel/AMD 64-bit)
- `goreflector-linux-arm64.tar.gz` (ARM 64-bit)
- `goreflector-linux-armv7.tar.gz` (ARM 32-bit v7)

**macOS:**
- `goreflector-darwin-amd64.tar.gz` (Intel Mac)
- `goreflector-darwin-arm64.tar.gz` (Apple Silicon M1/M2)

**Windows:**
- `goreflector-windows-amd64.exe.zip` (Intel/AMD 64-bit)
- `goreflector-windows-arm64.exe.zip` (ARM 64-bit)

**Artifacts:**
- Each binary is packaged with README.md and LICENSE
- SHA256 checksums generated for each artifact
- Combined `SHA256SUMS` file for all artifacts

**Process:**
1. Build binary for each platform/architecture
2. Package as `.tar.gz` (Unix) or `.zip` (Windows)
3. Generate SHA256 checksum
4. Upload to GitHub release
5. Create combined checksums file

## Creating a Release

### Option 1: GitHub UI

1. Go to https://github.com/maclarensg/goreflector/releases
2. Click "Draft a new release"
3. Create a new tag (e.g., `v1.0.0`)
4. Fill in release notes
5. Click "Publish release"
6. Wait for workflow to complete (~5-10 minutes)
7. Binaries will be attached to the release

### Option 2: GitHub CLI

```bash
# Create and publish release
gh release create v1.0.0 \
  --title "Release v1.0.0" \
  --notes "Release notes here"

# Or create draft release
gh release create v1.0.0 \
  --title "Release v1.0.0" \
  --notes "Release notes here" \
  --draft
```

### Option 3: Git Tags

```bash
# Create and push tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# Then create release from tag in GitHub UI
```

## Workflow Status Badges

Add to README.md:

```markdown
[![CI](https://github.com/maclarensg/goreflector/actions/workflows/ci.yml/badge.svg)](https://github.com/maclarensg/goreflector/actions/workflows/ci.yml)
[![Release](https://github.com/maclarensg/goreflector/actions/workflows/release.yml/badge.svg)](https://github.com/maclarensg/goreflector/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/maclarensg/goreflector/branch/main/graph/badge.svg)](https://codecov.io/gh/maclarensg/goreflector)
```

## Local Testing

Before pushing, run local checks:

```bash
# Run full CI pipeline locally
task ci

# Individual checks
task test
task lint
task gosec
task test:coverage
```

## Troubleshooting

### CI Fails on Coverage

```bash
# Check current coverage
task test:coverage

# View detailed coverage
go tool cover -html=coverage.out
```

Coverage must be ≥ 65% for CI to pass.

### Linter Fails

```bash
# Run linter locally
task lint

# Auto-fix issues
task lint:fix
```

### Security Scan Fails

```bash
# Run gosec locally
task gosec

# View detailed report
cat gosec-report.txt
```

### Release Workflow Fails

Common issues:
- **Upload fails**: Check GitHub token permissions
- **Build fails**: Test locally with `GOOS=linux GOARCH=amd64 go build`
- **Tag not found**: Ensure tag is pushed: `git push origin v1.0.0`

## Secrets Required

None currently. Workflows use `GITHUB_TOKEN` which is automatically provided.

## Future Enhancements

Planned workflow additions:
- [ ] Docker image build and push to GHCR
- [ ] Automated dependency updates (Dependabot)
- [ ] Nightly builds
- [ ] Performance benchmarking
- [ ] Integration tests against real services
- [ ] Automated changelog generation
- [ ] Slack/Discord notifications

## Workflow Permissions

The workflows require the following permissions:

**CI Workflow:**
- `contents: read` - Read repository contents
- `security-events: write` - Upload SARIF results

**Release Workflow:**
- `contents: write` - Upload release assets

## Cost Optimization

GitHub Actions usage:
- CI runs: ~2-3 minutes per run
- Release builds: ~5-10 minutes per release
- Free tier: 2,000 minutes/month (private repos)

All workflows use caching to minimize build times and costs.
