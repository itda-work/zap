---
number: 35
title: '상태 변경 명령어 통합: open/start/done/close → set'
state: done
labels:
    - enhancement
    - cli
assignees: []
created_at: "2026-01-19T00:54:49Z"
updated_at: "2026-01-19T01:06:09Z"
closed_at: "2026-01-19T01:06:09Z"
---

## 개요

상태를 변경하는 개별 명령어(open, start, done, close)를 `zap set <state> <number>` 하나로 통합

## 구현 완료

### 1. 새 명령어 `zap set <state> <number>`
- state 값: `open`, `wip`, `done`, `closed`
- `in-progress` 상태는 `wip`로 완전 대체

### 2. 제거된 명령어
- `zap open`, `zap start`, `zap done`, `zap close`

### 3. `fix-state` 명령 추가
- 잘못된 상태값(예: in-progress)을 가진 이슈를 찾아 수정
- `--dry-run`: 변경 없이 미리보기
- `--yes`: 모두 자동 수정

### 4. 수정된 파일
- `internal/cli/move.go`: set 명령 구현, 기존 명령 제거
- `internal/cli/fix_state.go`: fix-state 명령 신규
- `internal/issue/issue.go`: StateInProgress → StateWip
- `internal/cli/list.go`, `new.go`: 상태값 업데이트
- `internal/ai/prompt.go`: AI 프롬프트 업데이트
- `README.md`, `CLAUDE.md`: 문서 업데이트
- `internal/cli/init.go`: 에이전트 지침 템플릿 업데이트
- 테스트 파일들: 상태값 업데이트

## 사용 예시
```bash
zap set wip 1       # 작업 시작
zap set done 1      # 완료
zap set open 1      # 재오픈
zap set closed 1    # 취소/보류

zap fix-state       # 잘못된 상태 수정
```
