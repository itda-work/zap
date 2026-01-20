---
number: 40
title: zap list 명령에서 수정시각(상대시간) 출력
state: done
labels:
    - enhancement
assignees: []
created_at: "2026-01-20T05:28:10Z"
updated_at: "2026-01-20T05:30:37Z"
closed_at: "2026-01-20T05:30:37Z"
---

## 개요

`zap list` 명령 실행 시 각 이슈의 수정시각을 상대 시간 형식으로 표시합니다.

## 요구사항

- **형식**: 상대 시간 (예: 2시간 전, 3일 전, 1주 전)
- **위치**: 각 줄의 끝에 표시
- **기본값**: 항상 표시 (--no-date 플래그로 숨김 가능)

## 예시 출력

```
[wip]  #1    이슈 제목 [bug]     2시간 전
[open] #24   다른 이슈          3일 전
```

## 구현 범위

- printIssueList 함수 수정
- printMultiProjectIssueList 함수 수정
- --no-date 플래그 추가
- 상대 시간 포맷팅 함수 구현
