---
number: 26
title: repair 명령에 이슈 번호 충돌 감지/수정 기능 추가
state: done
labels:
    - enhancement
    - cli
assignees: []
created_at: 2026-01-18T10:46:50.170267+09:00
updated_at: 2026-01-18T11:03:07.160471+09:00
closed_at: 2026-01-18T11:03:07.160471+09:00
---

### 개요

`zap repair --conflicts` 옵션을 추가하여 이슈 번호 충돌을 감지하고 AI 기반으로 대화형 수정을 지원합니다.

### 배경

- 여러 사람이 동시에 `zap new` 실행 시 번호 충돌 가능
- AI(Claude 등)가 `zap new`를 사용하지 않고 수동으로 파일 생성 시 번호 오탐지
- 수동 파일 생성 시 기존 번호와 중복 발생

### 충돌 유형 (3가지)

1. **파일명 번호 중복**: `001-a.md`와 `001-b.md` 모두 존재
2. **frontmatter number 중복**: 서로 다른 파일이 같은 `number` 값을 가짐
3. **파일명-frontmatter 불일치**: `001-a.md`인데 frontmatter에 `number: 2`

### 수정 전략

- **나중에 생성된 파일**의 번호를 변경
- 생성 시점 판단 순서:
  1. `git log` (파일 최초 커밋 시점)
  2. frontmatter의 `created_at` 필드
- AI CLI (claude → codex → gemini 순) 사용하여 검증 후 수정

### 구현 요구사항

#### 새 옵션
- `zap repair --conflicts`: 번호 충돌 감지 및 수정 모드
- 기존 `--dry-run`, `--yes`, `--ai` 옵션과 호환

#### 기능 흐름
1. 충돌 감지 → 충돌 목록 표시
2. 각 충돌에 대해 변경 내용 표시 (파일명 변경, frontmatter 수정)
3. 사용자 확인 (대화형) 또는 `--yes`로 자동 진행
4. AI로 변경 내용 검증
5. 실제 수정 적용 (백업 생성)

#### 출력 예시
```
$ zap repair --conflicts
🔍 Checking for number conflicts...

Found 2 conflicts:

1. Duplicate filename number: 001
   - 001-feature-a.md (created: 2026-01-10)
   - 001-feature-b.md (created: 2026-01-15) ← will be renumbered to 026

2. Filename-frontmatter mismatch:
   - 003-bug-fix.md has number: 5 in frontmatter
   - Will update frontmatter to number: 3

Use --dry-run to preview changes without modifying files.
Proceed with repairs? [y/N]:
```

### 참고

- 기존 `repair` 명령: frontmatter 파싱 오류 수정 (`internal/cli/repair.go`)
- 번호 할당 로직: `findNextIssueNumber()` (`internal/cli/new.go:149`)
