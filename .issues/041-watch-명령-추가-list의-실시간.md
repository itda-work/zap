---
number: 41
title: watch 명령 추가 - list의 실시간 모니터링 버전
state: done
labels:
    - enhancement
assignees: []
created_at: "2026-01-20T09:46:53Z"
updated_at: "2026-01-20T09:49:39Z"
closed_at: "2026-01-20T09:49:39Z"
---

## 개요

list 명령의 실시간 모니터링 버전인 watch 명령을 추가합니다.

## 화면 구성

- 상단: 상태별 이슈 개수 (Open: N | WIP: N | Done: N | Closed: N)
- 하단: list 명령과 동일한 이슈 목록
- 마지막 갱신 시간 표시

## 갱신 방식

- fsnotify로 .issues/ 디렉토리 변경 감지 시 갱신

## 플랫폼 지원

- Windows, macOS, Linux 모두 지원

## 종료

- Ctrl+C로 종료
