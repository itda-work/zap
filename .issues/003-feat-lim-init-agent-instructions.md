---
number: 3
title: "feat: zap init - AI 에이전트용 지침 파일 생성 명령"
state: done
labels:
  - feature
  - cli
assignees:
  - allieus
created_at: 2026-01-15T18:39:24Z
updated_at: 2026-01-17T15:05:14Z
---

## 개요

`zap init <agent>` 명령을 추가하여 AI 코딩 어시스턴트용 지침 파일을 자동 생성합니다.

## 배경

- AI 코딩 어시스턴트(Claude, Codex, Gemini 등)가 .issues/ 디렉토리 구조와 zap 사용법을 이해하도록 지침 필요
- 프로젝트마다 수동으로 지침을 작성하는 것은 번거로움
- 표준화된 지침 템플릿으로 일관된 AI 어시스턴트 경험 제공

## 설계

### 명령어 형식

```bash
zap init <agent> [--path <path>]
```

### 지원 에이전트

| Agent | 기본 파일명 |
|-------|------------|
| claude | CLAUDE.md |
| codex | AGENTS.md |
| gemini | GEMINI.md |

### 옵션

- `--path <filepath>`: 지침 파일 경로 (기본: CLAUDE.md/AGENTS.md/GEMINI.md)

### 동작

1. 기본 경로: 프로젝트 루트 디렉토리
2. 파일이 이미 존재할 경우: 파일 끝에 추가 (append)
3. 지침 내용:
   - .issues/ 디렉토리 구조
   - zap CLI 명령어 사용법
   - 이슈 파일 형식 (YAML frontmatter)
   - 워크플로우 가이드

### 지침 템플릿 내용

```markdown
# Local Issue Management (zap)

## .issues/ 디렉토리 구조

.issues/
├── open/           # 새로 생성된 이슈
├── in-progress/    # 진행 중인 이슈
└── done/           # 완료된 이슈

## 이슈 파일 형식

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
---

이슈 본문 내용...

## zap 명령어

### 목록 조회
zap list                    # 열린 이슈 (open + in-progress)
zap list --all              # 전체 이슈
zap list --state open       # 특정 상태
zap list --label bug        # 레이블 필터

### 상세 보기
zap show 1                  # 이슈 #1 상세
zap show 1 --raw            # 원본 마크다운 출력

### 상태 변경
zap open 1                  # → open/
zap start 1                 # → in-progress/
zap done 1                  # → done/

### 검색
zap search "키워드"          # 제목/내용 검색
zap search --title "키워드"  # 제목만 검색

### 통계
zap stats                   # 상태별 이슈 수, 최근 활동

## 워크플로우

1. 새 이슈 생성: .issues/open/NNN-slug.md 파일 생성
2. 작업 시작: zap start <number>
3. 작업 완료: zap done <number>
```

## 작업 목록

- [x] internal/cli/init.go 파일 생성
- [x] 에이전트별 템플릿 정의
- [x] --path 옵션 구현 (파일 경로로 수정)
- [x] 기존 파일에 append 로직 구현
- [x] 테스트

## 진행 내역

### 2026-01-15

#### 구현 완료

1. **internal/cli/init.go 생성**
   - `zap init <agent>` 명령 구현
   - 지원 에이전트: claude, codex, gemini
   - 에이전트별 기본 파일명 매핑

2. **지침 템플릿**
   - .issues/ 디렉토리 구조
   - 이슈 파일 형식 (YAML frontmatter)
   - zap CLI 명령어 전체
   - TUI 단축키
   - 워크플로우 가이드

3. **--path 옵션**
   - 파일 경로 직접 지정 (폴더가 아닌 파일 경로)
   - 디렉토리 자동 생성
   - 예: `--path AI_GUIDE.md`, `--path docs/AGENTS.md`

4. **파일 존재 시 append**
   - 기존 파일 끝에 `---` 구분자 추가 후 지침 추가

**사용 예시:**

```bash
zap init claude                        # → CLAUDE.md
zap init claude --path AI_GUIDE.md     # → AI_GUIDE.md
zap init codex --path docs/AGENTS.md   # → docs/AGENTS.md (폴더 자동 생성)
```

## 참고

- 에이전트별 지침 파일 규약:
  - Claude Code: CLAUDE.md
  - OpenAI Codex CLI: AGENTS.md
  - Google Gemini: GEMINI.md
