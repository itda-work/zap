---
number: 52
title: '웹UI 제거: internal/web 패키지 및 관련 CLI 코드 삭제'
state: done
labels:
    - refactor
assignees: []
created_at: "2026-01-27T08:37:10Z"
updated_at: "2026-01-27T08:49:41Z"
closed_at: "2026-01-27T08:49:41Z"
---

## 배경

zap의 웹UI 기능(`internal/web/`)은 이슈를 브라우저에서 보기 위한 기능이지만,
PDCA 상태 모델 도입(#51) 전에 웹UI를 정리하는 것이 안전함.
웹UI에 새로운 상태(check, review)를 반영하는 대신, 웹UI 자체를 제거하여 유지보수 부담을 줄임.

## 제거 대상

1. `internal/web/` 패키지 전체
2. `zap serve` 명령 (start/stop/status/logs)
3. `zap show --web` 플래그
4. 관련 import 및 의존성

## 유지 대상

- `zap show` CLI 출력 (터미널 렌더링)
- `zap show --watch` (파일 감시)
- `zap show --refs` (참조 그래프)

## 선행 관계

- 이 이슈 완료 후 #51(PDCA 접목) 진행
