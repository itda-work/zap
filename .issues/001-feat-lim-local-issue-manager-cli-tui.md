---
number: 1
title: 'feat: zap - Local Issue Manager CLI/TUI 도구 개발'
state: done
labels:
    - feature
    - cli
    - tui
assignees:
    - allieus
created_at: 2026-01-15T18:39:24Z
updated_at: 2026-01-17T15:05:14Z
---

## 개요

`.issues/` 디렉토리 내 로컬 이슈를 관리하는 CLI/TUI 도구를 Go로 개발합니다.

## 배경

- 현재 `.issues/` 디렉토리의 이슈를 확인하려면 `tree`, `cat` 명령을 반복 사용해야 함
- 이슈 목록 조회, 상태 변경, 검색 등의 작업이 번거로움
- 개인 개발자 및 소규모 팀이 git으로 공유하며 사용할 수 있는 도구 필요

## 설계

### 프로젝트 구조

```
itda-issues/
├── cmd/
│   └── zap/
│       └── main.go
├── internal/
│   ├── issue/          # 이슈 파싱/관리 로직
│   │   ├── issue.go
│   │   ├── parser.go
│   │   └── store.go
│   ├── cli/            # CLI 명령어
│   │   ├── root.go
│   │   ├── list.go
│   │   ├── show.go
│   │   ├── move.go     # open/start/done/close
│   │   ├── search.go
│   │   └── stats.go
│   └── tui/            # TUI 컴포넌트
│       ├── app.go
│       ├── list.go
│       ├── detail.go
│       └── styles.go
├── .issues/
│   ├── open/
│   ├── in-progress/
│   ├── done/
│   └── closed/
├── go.mod
└── go.sum
```

### 상태 관리

디렉토리 기반 상태 관리:
- `open/` - 새로 생성된 이슈
- `in-progress/` - 진행 중인 이슈
- `done/` - 완료된 이슈
- `closed/` - 종료된 이슈 (완료 외 사유)

### 이슈 파일 형식

```markdown
---
number: 1
title: "이슈 제목"
state: open
labels:
  - bug
  - urgent
assignees:
  - username
created_at: 2026-01-15T18:39:24Z
updated_at: 2026-01-17T15:05:14Z
closed_at:
---

## 개요
이슈 내용...
```

### CLI 명령어

```bash
# 목록 조회
zap list                    # 열린 이슈 (open + in-progress)
zap list --all              # 전체 이슈
zap list --state open       # 특정 상태
zap list --label bug        # 레이블 필터

# 상세 보기
zap show 1                  # 이슈 #1 상세
zap show 1 --raw            # 원본 마크다운 출력

# 상태 변경
zap open 1                  # → open/
zap start 1                 # → in-progress/
zap done 1                  # → done/
zap close 1                 # → closed/

# 검색
zap search "키워드"          # 제목/내용 검색
zap search --title "키워드"  # 제목만 검색

# 통계
zap stats                   # 상태별 이슈 수, 최근 활동

# TUI
zap tui                     # TUI 모드 진입
zap                         # 인자 없으면 TUI 모드
```

### TUI 기능

- **읽기 전용** + 파일 변경 시 자동 새로고침 (fsnotify)
- **Bubble Tea + Lipgloss** 스택
- 이슈 목록 탐색, 상세 보기, 상태별 필터
- 키보드 단축키: j/k 이동, Enter 상세, q 종료, / 검색

### 의존성

```
github.com/spf13/cobra      # CLI 프레임워크
github.com/charmbracelet/bubbletea   # TUI 프레임워크
github.com/charmbracelet/lipgloss    # TUI 스타일링
github.com/charmbracelet/bubbles     # TUI 컴포넌트
github.com/fsnotify/fsnotify         # 파일 감시
gopkg.in/yaml.v3                     # YAML 파싱
```

## 작업 목록

### Phase 1: 프로젝트 초기화
- [x] Go 모듈 초기화 (`go mod init`)
- [x] 기본 디렉토리 구조 생성
- [x] 의존성 추가

### Phase 2: 핵심 로직 구현
- [x] 이슈 파싱 로직 (`internal/issue/parser.go`)
- [x] 이슈 저장소 로직 (`internal/issue/store.go`)
- [x] 이슈 모델 정의 (`internal/issue/issue.go`)

### Phase 3: CLI 구현
- [x] root 명령어 설정
- [x] `zap list` 구현
- [x] `zap show` 구현
- [x] `zap open/start/done` 구현
- [x] `zap search` 구현
- [x] `zap stats` 구현

### Phase 4: TUI 구현
- [x] TUI 앱 기본 구조
- [x] 이슈 목록 뷰
- [x] 이슈 상세 뷰
- [x] 파일 감시 및 자동 새로고침
- [x] 스타일링

### Phase 5: 마무리
- [x] 테스트 작성
- [x] README 작성
- [x] 빌드 및 설치 스크립트 (Makefile)

## 진행 내역

### 2026-01-15

#### Phase 5 완료: 테스트, README, Makefile

- 테스트 작성 (`internal/issue/issue_test.go`, `parser_test.go`)
- README.md 작성 (설치, 사용법, 파일 형식 문서화)
- Makefile 작성 (build, install, test, clean, cross-compile)
- closed 상태 제거 → open/in-progress/done 3가지 상태로 단순화
- 첫 커밋: `877c0ed`

#### 초기 구현 완료

**Phase 1-4 완료:**

1. **프로젝트 초기화**
   - `go mod init github.com/itda-work/zap`
   - 디렉토리 구조: `cmd/zap/`, `internal/{issue,cli,tui}/`

2. **핵심 로직** (`internal/issue/`)
   - `issue.go`: Issue 모델, State 타입 정의
   - `parser.go`: Markdown frontmatter 파싱/직렬화
   - `store.go`: 이슈 목록, 검색, 필터링, 통계

3. **CLI 구현** (`internal/cli/`)
   - `root.go`: 기본 설정, 인자 없으면 TUI 모드
   - `list.go`: 목록 조회 (--all, --state, --label, --assignee)
   - `show.go`: 상세 보기 (--raw)
   - `move.go`: 상태 변경 (open/start/done/close)
   - `search.go`: 키워드 검색 (--title)
   - `stats.go`: 통계 대시보드

4. **TUI 구현** (`internal/tui/`)
   - Bubble Tea + Lipgloss 스택
   - 이슈 목록/상세 뷰
   - 키보드 단축키: j/k 이동, Enter 상세, 1-4 상태필터, 0 전체, q 종료
   - fsnotify로 파일 변경 자동 감지 및 새로고침
   - 수동 새로고침: r 키

**테스트 결과:**
```bash
./zap list          # ✅ 동작
./zap show 1        # ✅ 동작
./zap stats         # ✅ 동작
./zap search "zap"  # ✅ 동작
./zap --help        # ✅ 동작
```

**의존성:**
- github.com/spf13/cobra v1.10.2
- github.com/charmbracelet/bubbletea v1.3.10
- github.com/charmbracelet/lipgloss v1.1.0
- github.com/charmbracelet/bubbles v0.21.0
- github.com/fsnotify/fsnotify v1.9.0
- gopkg.in/yaml.v3 v3.0.1

---

- 이슈 생성
- 요구사항 명확화 완료
  - CLI: Git 스타일 명령어
  - TUI: 읽기 전용 + 자동 새로고침
  - 상태 관리: 디렉토리 기반
  - 추가 기능: 검색/필터링, 통계/대시보드
