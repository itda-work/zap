---
number: 15
title: 'feat: zap list에 검색어 옵션 추가'
state: done
labels:
    - enhancement
    - cli
assignees: []
created_at: 2026-01-16T00:00:00Z
updated_at: 2026-01-16T00:00:00Z
---

## 개요

현재 `zap search <keyword>`는 별도 명령으로 존재하지만, `zap list`의 다른 필터 옵션들(`--state`, `--label`, `--assignee`)과 조합하여 사용할 수 없음.

`zap list`에 검색 옵션을 추가하여 필터 조합을 가능하게 함.

## 현재 상태

```bash
# 검색은 별도 명령
zap search "keyword"

# list 필터와 검색을 조합 불가
zap list --state open --label bug  # 검색 옵션 없음
```

## 제안

```bash
# 옵션으로 검색 추가
zap list --search "keyword"
zap list -S "keyword"

# 다른 필터와 조합 가능
zap list --state open --search "auth"
zap list --label bug --search "login" --assignee allieus
```

## 작업 목록

- [x] `--search`, `-S` 플래그 추가 (internal/cli/list.go)
- [x] 기존 필터들과 AND 조건으로 동작
- [x] `zap search` 명령 제거 (internal/cli/search.go 삭제)
- [x] `highlightKeyword` 함수 list.go로 이동
- [x] `--title-only` 옵션 추가
- [ ] 테스트 추가

## 고려사항

- 검색은 제목+본문 기본, `--title-only` 옵션으로 제목만 검색 지원
- `highlightKeyword` 함수는 list.go로 이동

## 구현 기록

### 2026-01-16

- `--search`, `-S` 플래그 추가
- `--title-only` 플래그 추가 (제목만 검색)
- `filterBySearch()` 함수 구현
- `highlightKeyword()` 함수 search.go에서 list.go로 이동
- `printIssueList()` 함수에 키워드 하이라이트 기능 추가
- `internal/cli/search.go` 삭제
