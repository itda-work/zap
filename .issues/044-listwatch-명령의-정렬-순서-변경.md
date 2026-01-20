---
number: 44
title: 'list/watch 명령의 정렬 순서 변경: 상태별 그룹화'
state: done
labels: []
assignees: []
created_at: "2026-01-20T13:12:28Z"
updated_at: "2026-01-20T13:16:41Z"
closed_at: "2026-01-20T13:16:41Z"
---

## 배경

현재 `list`와 `watch` 명령에서 이슈 정렬이 `CreatedAt` 기준으로 되어 있고, recently closed 이슈가 목록 마지막에 추가됩니다. 이로 인해 방금 완료한 이슈가 최하단에 위치하는 문제가 있습니다.

## 문제

- 정렬 기준이 `CreatedAt`인데, 워크플로우 관점에서는 `UpdatedAt`이 더 적합
- recently closed 이슈가 단순 append되어 항상 마지막에 위치
- 완료된 작업을 바로 확인하기 어려움

## 목표

상태별 그룹화 + UpdatedAt 역순 정렬로 변경:

1. **그룹 순서**: done → closed → wip → open
2. **각 그룹 내**: UpdatedAt 역순 (최근 업데이트가 먼저)
3. **done/closed 필터**: 최근 5분 이내 업데이트된 것만 표시 (기본값)

### 예시 출력
```
[done]   #4   방금 완료한 이슈       2 minutes ago
[closed] #6   방금 닫은 이슈         4 minutes ago
[wip]    #5   작업 중인 이슈 A       3 minutes ago
[wip]    #3   작업 중인 이슈 B       1 hour ago
[open]   #7   대기 중인 이슈 C       5 minutes ago
[open]   #2   대기 중인 이슈 D       2 days ago
```

## 비목표

- `--all` 플래그 동작 변경 (전체 이슈 표시는 그대로 유지)
- 특정 상태 필터(`--state`) 사용 시 동작 변경

## 제약사항

- 기존 CLI 인터페이스 호환성 유지
- recentClosedDuration 기본값을 5분으로 변경
