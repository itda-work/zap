---
number: 24
title: 'feat: zap new 명령 및 표준 이슈 포맷 구현'
state: done
labels:
    - enhancement
    - completed
assignees: []
created_at: 2026-01-17T17:06:04.016522+09:00
updated_at: 2026-01-17T17:07:04.525375+09:00
closed_at: 2026-01-17T17:07:04.525375+09:00
---

## 개요

AI들이 이슈 생성 시 포맷이 조금씩 다르게 생성되어 `zap list` 명령에서 파싱 실패가 발생하는 문제를 해결하기 위해 `zap new` 명령과 표준 이슈 포맷을 구현했습니다.

## 구현 내용

### 1. `internal/cli/new.go` 생성
- `zap new <title>` 명령 구현
- 플래그:
  - `-l, --label`: 레이블 추가 (반복 가능)
  - `-a, --assignee`: 담당자 추가 (반복 가능)
  - `-b, --body`: 이슈 본문 내용
  - `-e, --editor`: 에디터로 본문 작성
  - `-s, --state`: 초기 상태 (기본: open)
- 주요 함수:
  - `generateSlug()`: 한글 지원 URL-friendly slug 생성
  - `findNextIssueNumber()`: 다음 이슈 번호 계산 (파싱 실패 파일도 고려)
  - `extractNumberFromFilename()`: 파일명에서 번호 추출
  - `openEditor()`: \$EDITOR/\$VISUAL 환경변수로 에디터 실행
- stdin 파이프 입력 지원 (AI 통합용)

### 2. `internal/cli/new_test.go` 생성
- `TestGenerateSlug`: 12개 테스트 케이스
- `TestExtractNumberFromFilename`: 7개 테스트 케이스
- `TestFindNextIssueNumber`: 4개 테스트 케이스
- `TestNewCommandIntegration`: 통합 테스트

### 3. `internal/cli/init.go` 수정
- "이슈 생성 (중요!)" 섹션 추가
- `zap new` 사용 강조
- 수동 생성 시 검증 체크리스트 제공
- 워크플로우 업데이트

## 표준 이슈 포맷

### 파일명 규칙
`NNN-slug.md` (예: `024-feat-user-auth.md`)

### Frontmatter 형식
```yaml
---
number: 24
title: "이슈 제목"
state: open
labels:
  - enhancement
assignees:
  - username
created_at: 2026-01-17T12:00:00Z
updated_at: 2026-01-17T12:00:00Z
---
```

## 테스트 결과

- `go build ./...` ✅
- `go test ./...` ✅
- 수동 테스트 ✅
  - 기본 이슈 생성
  - 한글 제목 이슈 생성
  - 레이블/담당자 추가
  - 파이프 본문 입력
  - `zap list` 파싱 확인
