# Homebrew 및 winget 패키지 등록 가이드

zap CLI를 Homebrew(macOS/Linux)와 winget(Windows) 패키지 관리자에 등록하는 방법을 설명합니다.

---

## 1. Homebrew 등록

Homebrew는 macOS 및 Linux에서 사용되는 패키지 관리자입니다.

### 1.1 자체 tap 생성 및 운영

공식 homebrew-core에 등록하기 전, 또는 독립적으로 배포하려면 자체 tap을 운영합니다.

#### 1.1.1 homebrew-tap 저장소 생성

GitHub에 `homebrew-tap` 저장소를 생성합니다.

```bash
# 저장소 이름은 반드시 homebrew-* 형식이어야 함
# 예: itda-work/homebrew-tap
```

#### 1.1.2 Formula 작성 (zap.rb)

`Formula/zap.rb` 파일을 생성합니다:

```ruby
class Zap < Formula
  desc "Local issue management CLI tool"
  homepage "https://github.com/itda-work/zap"
  version "0.1.0"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/itda-work/zap/releases/download/v#{version}/zap-macos-amd64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"

      def install
        bin.install "zap-macos-amd64" => "zap"
      end
    end

    on_arm do
      url "https://github.com/itda-work/zap/releases/download/v#{version}/zap-macos-arm64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"

      def install
        bin.install "zap-macos-arm64" => "zap"
      end
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/itda-work/zap/releases/download/v#{version}/zap-linux-amd64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"

      def install
        bin.install "zap-linux-amd64" => "zap"
      end
    end

    on_arm do
      url "https://github.com/itda-work/zap/releases/download/v#{version}/zap-linux-arm64"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"

      def install
        bin.install "zap-linux-arm64" => "zap"
      end
    end
  end

  test do
    system "#{bin}/zap", "--version"
  end
end
```

#### 1.1.3 SHA256 체크섬 계산

릴리즈된 바이너리의 체크섬을 계산합니다:

```bash
# 로컬 파일
sha256sum zap-macos-arm64

# 또는 URL에서 직접
curl -sL https://github.com/itda-work/zap/releases/download/v0.1.0/zap-macos-arm64 | sha256sum
```

#### 1.1.4 GitHub Actions 자동화

릴리즈 시 Formula를 자동으로 업데이트하는 워크플로우:

```yaml
# .github/workflows/homebrew.yml
name: Update Homebrew Formula

on:
  release:
    types: [published]

jobs:
  homebrew:
    runs-on: ubuntu-latest
    steps:
      - name: Update Homebrew formula
        uses: dawidd6/action-homebrew-bump-formula@v3
        with:
          token: ${{ secrets.HOMEBREW_TAP_TOKEN }}
          tap: itda-work/homebrew-tap
          formula: zap
          tag: ${{ github.event.release.tag_name }}
```

또는 수동으로 업데이트하는 스크립트:

```bash
#!/bin/bash
# scripts/update-homebrew.sh

VERSION=$1
TAP_REPO="itda-work/homebrew-tap"

# 체크섬 다운로드
CHECKSUMS=$(curl -sL "https://github.com/itda-work/zap/releases/download/v${VERSION}/checksums.txt")

# 각 플랫폼 체크섬 추출
SHA_MACOS_AMD64=$(echo "$CHECKSUMS" | grep "zap-macos-amd64" | awk '{print $1}')
SHA_MACOS_ARM64=$(echo "$CHECKSUMS" | grep "zap-macos-arm64" | awk '{print $1}')
SHA_LINUX_AMD64=$(echo "$CHECKSUMS" | grep "zap-linux-amd64" | awk '{print $1}')
SHA_LINUX_ARM64=$(echo "$CHECKSUMS" | grep "zap-linux-arm64" | awk '{print $1}')

echo "Version: $VERSION"
echo "macOS amd64: $SHA_MACOS_AMD64"
echo "macOS arm64: $SHA_MACOS_ARM64"
echo "Linux amd64: $SHA_LINUX_AMD64"
echo "Linux arm64: $SHA_LINUX_ARM64"
```

#### 1.1.5 사용자 설치 방법

```bash
# tap 추가
brew tap itda-work/tap

# 설치
brew install zap

# 업그레이드
brew upgrade zap
```

### 1.2 homebrew-core 공식 등록

#### 1.2.1 등록 조건

homebrew-core에 등록하려면 다음 조건을 충족해야 합니다:

- **Notable**: 프로젝트가 "주목할 만한" 수준이어야 함
  - GitHub stars 75개 이상 (권장)
  - 또는 다른 방식으로 인지도 증명
- **Stable release**: 안정적인 릴리즈 버전 존재
- **Open source license**: MIT, Apache 2.0 등 승인된 라이선스
- **No vendored dependencies**: 가능한 시스템 라이브러리 사용
- **Build from source**: 바이너리가 아닌 소스에서 빌드

#### 1.2.2 소스 빌드용 Formula

homebrew-core는 소스에서 빌드하는 것을 선호합니다:

```ruby
class Zap < Formula
  desc "Local issue management CLI tool"
  homepage "https://github.com/itda-work/zap"
  url "https://github.com/itda-work/zap/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "REPLACE_WITH_SOURCE_TARBALL_SHA256"
  license "MIT"

  depends_on "go" => :build

  def install
    ldflags = %W[
      -s -w
      -X github.com/itda-work/zap/internal/cli.Version=#{version}
      -X github.com/itda-work/zap/internal/cli.BuildDate=#{time.iso8601}
    ]
    system "go", "build", *std_go_args(ldflags:), "./cmd/zap"

    generate_completions_from_executable(bin/"zap", "completion")
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/zap --version")
  end
end
```

#### 1.2.3 PR 제출 절차

1. **homebrew-core 포크**:
   ```bash
   gh repo fork Homebrew/homebrew-core --clone
   cd homebrew-core
   ```

2. **Formula 생성**:
   ```bash
   brew create https://github.com/itda-work/zap/archive/refs/tags/v0.1.0.tar.gz --go
   # 또는 수동으로 Formula/z/zap.rb 생성
   ```

3. **로컬 테스트**:
   ```bash
   brew install --build-from-source ./Formula/z/zap.rb
   brew test zap
   brew audit --strict --new zap
   ```

4. **PR 제출**:
   ```bash
   git checkout -b zap-0.1.0
   git add Formula/z/zap.rb
   git commit -m "zap 0.1.0 (new formula)"
   gh pr create --title "zap 0.1.0 (new formula)" --body "..."
   ```

#### 1.2.4 PR 템플릿

```markdown
## Description

zap is a local issue management CLI tool that manages issues in `.issues/` directory.

## Checklist

- [x] Have you followed the [guidelines for contributing](https://github.com/Homebrew/homebrew-core/blob/HEAD/CONTRIBUTING.md)?
- [x] Have you checked that there aren't other open [pull requests](https://github.com/Homebrew/homebrew-core/pulls) for the same formula update/change?
- [x] Have you built your formula locally with `brew install --build-from-source <formula>`?
- [x] Does your build pass `brew audit --strict <formula>`?
```

---

## 2. winget 등록

winget은 Windows 10/11의 공식 패키지 관리자입니다.

### 2.1 자체 manifest 저장소 운영

#### 2.1.1 manifest 파일 구조

winget manifest는 3개의 YAML 파일로 구성됩니다:

```
manifests/
└── i/
    └── itda-work/
        └── zap/
            └── 0.1.0/
                ├── itda-work.zap.yaml           # 버전 manifest
                ├── itda-work.zap.installer.yaml # 설치 정보
                └── itda-work.zap.locale.en-US.yaml # 로케일 정보
```

#### 2.1.2 버전 manifest (itda-work.zap.yaml)

```yaml
PackageIdentifier: itda-work.zap
PackageVersion: 0.1.0
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.6.0
```

#### 2.1.3 설치 manifest (itda-work.zap.installer.yaml)

```yaml
PackageIdentifier: itda-work.zap
PackageVersion: 0.1.0
Platform:
  - Windows.Desktop
MinimumOSVersion: 10.0.0.0
InstallerType: portable
Commands:
  - zap
Installers:
  - Architecture: x64
    InstallerUrl: https://github.com/itda-work/zap/releases/download/v0.1.0/zap-windows-amd64.exe
    InstallerSha256: REPLACE_WITH_ACTUAL_SHA256
    NestedInstallerType: portable
    NestedInstallerFiles:
      - RelativeFilePath: zap-windows-amd64.exe
        PortableCommandAlias: zap
  - Architecture: arm64
    InstallerUrl: https://github.com/itda-work/zap/releases/download/v0.1.0/zap-windows-arm64.exe
    InstallerSha256: REPLACE_WITH_ACTUAL_SHA256
    NestedInstallerType: portable
    NestedInstallerFiles:
      - RelativeFilePath: zap-windows-arm64.exe
        PortableCommandAlias: zap
ManifestType: installer
ManifestVersion: 1.6.0
```

#### 2.1.4 로케일 manifest (itda-work.zap.locale.en-US.yaml)

```yaml
PackageIdentifier: itda-work.zap
PackageVersion: 0.1.0
PackageLocale: en-US
Publisher: itda-work
PublisherUrl: https://github.com/itda-work
PackageName: zap
PackageUrl: https://github.com/itda-work/zap
License: MIT
LicenseUrl: https://github.com/itda-work/zap/blob/main/LICENSE
ShortDescription: Local issue management CLI tool
Description: zap is a CLI tool that manages local issues in .issues/ directory
Tags:
  - cli
  - issue-tracker
  - productivity
  - developer-tools
ManifestType: defaultLocale
ManifestVersion: 1.6.0
```

#### 2.1.5 로컬 테스트

```powershell
# winget validate로 manifest 검증
winget validate --manifest .\manifests\i\itda-work\zap\0.1.0\

# 로컬 manifest로 설치 테스트
winget install --manifest .\manifests\i\itda-work\zap\0.1.0\
```

### 2.2 winget-pkgs 공식 등록

#### 2.2.1 wingetcreate 도구 활용

Microsoft 공식 도구로 manifest를 자동 생성합니다:

```powershell
# 설치
winget install wingetcreate

# 새 manifest 생성
wingetcreate new https://github.com/itda-work/zap/releases/download/v0.1.0/zap-windows-amd64.exe

# 기존 manifest 업데이트
wingetcreate update itda-work.zap --version 0.2.0 --urls https://github.com/itda-work/zap/releases/download/v0.2.0/zap-windows-amd64.exe
```

#### 2.2.2 PR 제출 절차

1. **winget-pkgs 포크**:
   ```powershell
   gh repo fork microsoft/winget-pkgs --clone
   cd winget-pkgs
   ```

2. **manifest 생성**:
   ```powershell
   # wingetcreate로 생성 후 manifests/ 디렉토리에 복사
   wingetcreate new --out .\manifests https://github.com/itda-work/zap/releases/download/v0.1.0/zap-windows-amd64.exe
   ```

3. **검증**:
   ```powershell
   # manifest 유효성 검사
   winget validate --manifest .\manifests\i\itda-work\zap\0.1.0\
   ```

4. **PR 제출**:
   ```powershell
   git checkout -b itda-work.zap-0.1.0
   git add manifests/
   git commit -m "New package: itda-work.zap version 0.1.0"
   gh pr create
   ```

#### 2.2.3 자동 업데이트 설정 (komac)

[komac](https://github.com/russellbanks/Komac)은 winget manifest 자동 업데이트 도구입니다:

```yaml
# .github/workflows/winget.yml
name: Update winget manifest

on:
  release:
    types: [published]

jobs:
  winget:
    runs-on: windows-latest
    steps:
      - name: Install komac
        run: winget install komac

      - name: Update manifest
        run: |
          komac update itda-work.zap --version ${{ github.event.release.tag_name }} `
            --urls "https://github.com/itda-work/zap/releases/download/${{ github.event.release.tag_name }}/zap-windows-amd64.exe" `
            --submit
        env:
          GITHUB_TOKEN: ${{ secrets.WINGET_TOKEN }}
```

또는 GitHub Actions에서 직접 komac 사용:

```yaml
# .github/workflows/winget.yml
name: Publish to winget

on:
  release:
    types: [released]

jobs:
  publish:
    runs-on: windows-latest
    steps:
      - name: Submit to winget
        uses: vedantmgoyal9/winget-releaser@main
        with:
          identifier: itda-work.zap
          installers-regex: '\.exe$'
          token: ${{ secrets.WINGET_TOKEN }}
```

---

## 3. 릴리즈 자동화 연동

### 3.1 통합 릴리즈 워크플로우

Homebrew와 winget 업데이트를 통합한 워크플로우:

```yaml
# .github/workflows/release.yml (기존 파일에 추가)
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    outputs:
      version: ${{ steps.version.outputs.version }}
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Get version
        id: version
        run: echo "version=${GITHUB_REF_NAME#v}" >> $GITHUB_OUTPUT

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Run tests
        run: go test -v ./...

      - name: Build binaries
        run: |
          VERSION=${GITHUB_REF_NAME}
          BUILD_DATE=$(date -u +%Y-%m-%d)
          LDFLAGS="-s -w -X github.com/itda-work/zap/internal/cli.Version=${VERSION} -X github.com/itda-work/zap/internal/cli.BuildDate=${BUILD_DATE}"

          GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o dist/zap-linux-amd64 ./cmd/zap
          GOOS=linux GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o dist/zap-linux-arm64 ./cmd/zap
          GOOS=darwin GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o dist/zap-macos-amd64 ./cmd/zap
          GOOS=darwin GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o dist/zap-macos-arm64 ./cmd/zap
          GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS}" -o dist/zap-windows-amd64.exe ./cmd/zap
          GOOS=windows GOARCH=arm64 go build -ldflags="${LDFLAGS}" -o dist/zap-windows-arm64.exe ./cmd/zap

      - name: Create checksums
        run: |
          cd dist
          sha256sum * > checksums.txt

      - name: Create Release
        uses: softprops/action-gh-release@v2
        with:
          files: |
            dist/*
          generate_release_notes: true

  homebrew:
    needs: release
    runs-on: ubuntu-latest
    steps:
      - name: Update Homebrew tap
        uses: dawidd6/action-homebrew-bump-formula@v3
        with:
          token: ${{ secrets.HOMEBREW_TAP_TOKEN }}
          tap: itda-work/homebrew-tap
          formula: zap
          tag: ${{ github.ref_name }}

  winget:
    needs: release
    runs-on: windows-latest
    steps:
      - name: Submit to winget
        uses: vedantmgoyal9/winget-releaser@main
        with:
          identifier: itda-work.zap
          installers-regex: '\.exe$'
          token: ${{ secrets.WINGET_TOKEN }}
```

### 3.2 필요한 Secrets

| Secret | 용도 | 생성 방법 |
|--------|------|----------|
| `HOMEBREW_TAP_TOKEN` | homebrew-tap 저장소 접근 | GitHub PAT (repo 권한) |
| `WINGET_TOKEN` | winget-pkgs PR 생성 | GitHub PAT (public_repo 권한) |

### 3.3 체크섬 관리

현재 release.yml에서 이미 `checksums.txt`를 생성하고 있습니다:

```bash
cd dist
sha256sum * > checksums.txt
```

이 파일은 릴리즈 에셋으로 업로드되며, Homebrew Formula나 winget manifest 업데이트 시 활용됩니다.

---

## 4. 체크리스트

### Homebrew 자체 tap 등록

- [ ] `itda-work/homebrew-tap` 저장소 생성
- [ ] `Formula/zap.rb` 작성
- [ ] 체크섬 계산 및 적용
- [ ] `brew tap itda-work/tap && brew install zap` 테스트
- [ ] GitHub Actions 자동화 설정 (선택)

### Homebrew-core 공식 등록

- [ ] 등록 조건 확인 (stars 75+, stable release 등)
- [ ] 소스 빌드용 Formula 작성
- [ ] `brew audit --strict --new zap` 통과
- [ ] homebrew-core PR 제출

### winget 자체 manifest 등록

- [ ] manifest 3개 파일 작성
- [ ] `winget validate` 통과
- [ ] 로컬 설치 테스트

### winget-pkgs 공식 등록

- [ ] wingetcreate로 manifest 생성
- [ ] `winget validate` 통과
- [ ] winget-pkgs PR 제출
- [ ] komac 자동화 설정 (선택)

---

## 참고 자료

- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Homebrew Acceptable Formulae](https://docs.brew.sh/Acceptable-Formulae)
- [winget-pkgs 기여 가이드](https://github.com/microsoft/winget-pkgs/blob/master/CONTRIBUTING.md)
- [wingetcreate 문서](https://github.com/microsoft/winget-create)
- [komac 문서](https://github.com/russellbanks/Komac)
