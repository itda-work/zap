---
number: 29
title: 'state: in-progress를 wip로 변경'
state: done
labels:
    - enhancement
assignees: []
created_at: 2026-01-18T19:02:00.677461+09:00
updated_at: 2026-01-18T19:16:14.149478+09:00
closed_at: 2026-01-18T19:16:14.149479+09:00
---

## 개요

`in-progress` 상태를 `wip`로 변경하여 더 간결한 상태명을 사용합니다.

## 요구사항

### 1. 상태명 변경
- `in-progress` → `wip`

### 2. 하위 호환성
- 기존 `in-progress` 값을 읽을 때 정상적으로 인식
- `in-progress`와 `wip` 모두 동일한 상태로 처리

### 3. 출력 정규화
- 목록, 상세보기, 통계 등 모든 출력에서 `wip`로 표시
- 내부 값이 `in-progress`여도 `wip`로 표시

### 4. 쓰기 동작
- `zap start` 명령 실행 시 `state: wip`로 저장
- 새로운 상태 변경은 항상 `wip` 사용

### 5. 점진적 마이그레이션
- 기존 `in-progress` 파일은 자동 변환하지 않음
- 사용자가 해당 이슈의 상태를 변경할 때 `wip`로 마이그레이션
- 별도의 마이그레이션 명령 불필요

## 영향 범위

- `zap list`: 출력 시 wip로 표시
- `zap show`: 출력 시 wip로 표시
- `zap start`: state: wip로 저장
- `zap stats`: wip 상태로 집계
- 파싱 로직: in-progress와 wip 모두 인식

## 작업 내역

### Web UI 변경 (완료)

**수정 파일:**

1. `internal/issue/issue.go:60` - `ParseState`에서 "wip" 인식 추가
2. `internal/web/templates.go` - `displayState` 헬퍼 함수 추가
3. `internal/web/templates/dashboard.html` - 통계 카드, 필터 탭, 이슈 목록에서 wip 표시
4. `internal/web/templates/issue.html` - 이슈 상세 페이지에서 wip 표시
5. `internal/issue/issue_test.go` - "wip" 테스트 케이스 추가

**변경 내용:**

- 대시보드 통계 카드 라벨: `In Progress` → `WIP`
- 필터 탭 URL: `/?state=in-progress` → `/?state=wip`
- 필터 탭 라벨: `In Progress` → `WIP`
- 상태 표시: `displayState` 함수로 `in-progress` → `wip` 변환
