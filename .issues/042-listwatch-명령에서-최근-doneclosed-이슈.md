---
number: 42
title: list/watch 명령에서 최근 done/closed 이슈 일정 시간 표시
state: done
labels:
    - enhancement
assignees: []
created_at: "2026-01-20T10:03:19Z"
updated_at: "2026-01-20T10:14:44Z"
closed_at: "2026-01-20T10:14:44Z"
---

## 개요

`list`, `watch` 명령에서 방금 done이나 closed로 변경된 이슈를 바로 목록에서 제거하지 않고, 일정 시간 동안 표시하여 상태 변경을 명시적으로 보여주는 기능입니다.

## 요구사항

### 동작 방식
- done이나 closed 상태로 변경된 이슈가 `updated_at` 기준 설정된 시간(기본 5분) 이내인 경우:
  - 목록에 계속 표시
  - fg/bg 색상을 반전하여 눈에 띄게 강조
- 설정된 시간이 지나면 목록에서 제외

### 설정
- 유지 시간을 환경변수나 설정 파일로 조정 가능하게 구현

## 적용 대상
- `zap list` 명령
- `zap watch` 명령
