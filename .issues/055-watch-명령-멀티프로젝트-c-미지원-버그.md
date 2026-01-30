---
number: 55
title: 'fix: watch 명령 멀티프로젝트(-C) 미지원 버그'
state: done
labels:
    - bug
    - cli
assignees: []
created_at: "2026-01-30T04:55:10Z"
updated_at: "2026-01-30T04:57:05Z"
closed_at: "2026-01-30T04:57:05Z"
---

## 개요

`zap watch` 명령이 멀티프로젝트 모드(`-C` 플래그 복수 지정)를 지원하지 않는다.
`-C ../skills.scheduler -C .` 처럼 2개 이상 프로젝트를 지정해도 1개 프로젝트 이슈만 표시된다.

## 원인

`watch.go:runWatch()`가 `isMultiProjectMode()` 분기 없이 항상 `getIssuesDir()`로 싱글 프로젝트 경로만 사용.
`list` 명령은 `runMultiProjectList()` 분기가 있지만, `watch`에는 동일 처리가 누락되어 있다.

## 수정 방향

`list.go`의 멀티프로젝트 패턴을 `watch.go`에 동일 적용:
1. `runWatch()` 진입 시 `isMultiProjectMode()` 체크
2. 멀티프로젝트이면 `getMultiStore()`로 `MultiStore` 생성
3. `renderWatch()`에서 `multiStore.ListAll()`로 전체 프로젝트 이슈 조회
4. `printWatchIssueList`에서 프로젝트 prefix 표시 (list.go의 printMultiProjectIssueList 패턴)
5. fsnotify watcher에 모든 프로젝트의 .issues/ 디렉토리 등록

## 참고 파일
- internal/cli/watch.go (수정 대상)
- internal/cli/list.go (runMultiProjectList 패턴 참고)
- internal/cli/root.go (isMultiProjectMode, getMultiStore)
- internal/project/multistore.go (MultiStore.ListAll)
