---
number: 43
title: 'watch: add 1-minute auto-refresh and use English ago labels'
state: done
labels:
    - enhancement
assignees: []
created_at: "2026-01-20T10:55:41Z"
updated_at: "2026-01-20T10:57:09Z"
closed_at: "2026-01-20T10:57:09Z"
---

## 문제

1. `watch` 명령이 파일 변경 시에만 화면을 갱신함
2. `done` 상태로 전환 후 몇 분이 지나도 상대 시간이 계속 "방금" 상태로 남아있음
3. 상대 시간 레이블이 한국어로 되어 있음

## 해결 방안

### 1. 1분 주기 자동 갱신 추가
- 기존: fsnotify로 `.issues/` 디렉토리 변경 시에만 갱신
- 변경: 파일 변경 감지 + 1분마다 주기적 갱신

### 2. 상대 시간 레이블 영어화
- `방금` → `just now`
- `N분 전` → `N min ago`
- `N시간 전` → `N hr ago`
- `N일 전` → `N day ago` / `N days ago`
- `N주 전` → `N week ago` / `N weeks ago`
- `N개월 전` → `N month ago` / `N months ago`
- `N년 전` → `N year ago` / `N years ago`

## 수정 대상 파일

- `internal/cli/watch.go`: ticker 추가
- `internal/cli/utils.go`: `formatRelativeTime()` 함수 영어화
