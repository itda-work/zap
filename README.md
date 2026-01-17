# zap - Local Issue Manager

`.issues/` 디렉토리 내 로컬 이슈를 관리하는 CLI 도구입니다.

## 설치

### One-Line 설치 (권장)

**macOS / Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/itda-work/zap/main/scripts/install.sh | bash
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/itda-work/zap/main/scripts/install.ps1 | iex
```

### Go로 설치

```bash
go install github.com/itda-work/zap/cmd/zap@latest
```

### 소스에서 빌드

```bash
git clone https://github.com/itda-work/zap.git
cd zap
make install
```

## 사용법

```bash
# 이슈 목록
zap list                    # 활성 이슈 (open + in-progress)
zap list --all              # 전체 이슈
zap list --state done       # 특정 상태
zap list --label bug        # 레이블 필터

# 이슈 상세
zap show 1                  # 이슈 #1 상세
zap show 1 --raw            # 원본 마크다운

# 상태 변경
zap open 1                  # → open/
zap start 1                 # → in-progress/
zap done 1                  # → done/
zap close 1                 # → closed/ (취소/보류)

# 검색 & 통계
zap search "키워드"          # 제목/내용 검색
zap stats                   # 통계 대시보드

# 다른 프로젝트 이슈 관리 (-C 옵션)
zap -C ~/other-project list         # 다른 프로젝트 이슈 목록
zap -C ~/other-project show 5       # 다른 프로젝트 이슈 상세
zap -C ~/other-project start 5      # 다른 프로젝트 이슈 상태 변경

# AI 에이전트 지침 파일 생성
zap init claude             # CLAUDE.md 생성
zap init codex              # AGENTS.md 생성
zap init gemini             # GEMINI.md 생성
zap init claude --path AI_GUIDE.md  # 지정 파일에 생성
```

## 이슈 파일 형식

`.issues/` 디렉토리 구조:

```
.issues/
├── open/           # 새로 생성된 이슈
├── in-progress/    # 진행 중
├── done/           # 완료
└── closed/         # 취소/보류
```

이슈 파일 (`001-feature-name.md`):

```markdown
---
number: 1
title: "이슈 제목"
state: open
labels:
  - feature
  - cli
assignees:
  - username
created_at: 2026-01-15T00:00:00Z
updated_at: 2026-01-15T00:00:00Z
---

## 개요

이슈 내용...

## 작업 목록

- [ ] 작업 1
- [x] 작업 2

## 진행 내역

### 2026-01-15

- 작업 내용
```

## 셸 자동완성

이슈 번호 등의 인자에 대해 Tab 키 자동완성을 지원합니다.

### Bash

```bash
# 현재 세션
source <(zap completion bash)

# 영구 적용 (Linux)
zap completion bash > /etc/bash_completion.d/zap

# 영구 적용 (macOS + Homebrew)
zap completion bash > $(brew --prefix)/etc/bash_completion.d/zap
```

### Zsh

```zsh
# 현재 세션
source <(zap completion zsh)

# 영구 적용
echo 'source <(zap completion zsh)' >> ~/.zshrc
```

### Fish

```fish
zap completion fish | source

# 영구 적용
zap completion fish > ~/.config/fish/completions/zap.fish
```

### PowerShell

```powershell
# 현재 세션
zap completion powershell | Out-String | Invoke-Expression

# 영구 적용
zap completion powershell >> $PROFILE
```

### Windows 주의사항

- **PowerShell**: 자동완성 지원 ✓
- **cmd.exe**: 지원하지 않음 ✗ (동적 자동완성 메커니즘 없음)

Windows에서는 **PowerShell 사용을 권장**합니다.

## 개발

```bash
# 의존성 설치
go mod download

# 빌드
make build

# 테스트
make test

# 로컬 설치
make install
```

## 라이선스

MIT
