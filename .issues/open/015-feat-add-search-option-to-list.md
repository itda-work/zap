---
number: 15
title: "feat: zap list에 검색어 옵션 추가"
state: open
labels:
  - enhancement
  - cli
assignees: []
created_at: 2026-01-16T00:00:00Z
updated_at: 2026-01-16T00:00:00Z
closed_at:
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

- [ ] `--search`, `-S` 플래그 추가 (internal/cli/list.go)
- [ ] 기존 필터들과 AND 조건으로 동작
- [ ] `zap search` 명령 제거 (internal/cli/search.go 삭제)
- [ ] 테스트 추가

## 고려사항

- 검색은 제목+본문 기본, `--title-only` 옵션으로 제목만 검색 지원
- `highlightKeyword` 함수는 list.go로 이동
