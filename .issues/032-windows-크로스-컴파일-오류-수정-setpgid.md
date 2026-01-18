---
number: 32
title: 'fix: Windows 크로스 컴파일 오류 수정 (Setpgid)'
state: done
labels:
    - bug
    - platform
assignees: []
created_at: 2026-01-19T08:25:45.545323+09:00
updated_at: 2026-01-19T08:25:56.719022+09:00
closed_at: 2026-01-19T08:25:56.719022+09:00
---

## 문제
Windows 크로스 컴파일 시 `syscall.SysProcAttr.Setpgid` 필드가 존재하지 않아 빌드 실패.

```
internal/cli/serve.go:226:3: unknown field Setpgid in struct literal of type syscall.SysProcAttr
```

## 원인
`Setpgid`는 Unix 전용 필드로, Windows의 `syscall.SysProcAttr`에는 존재하지 않음.

## 해결 방법
플랫폼별 빌드 파일로 분리:
- `serve_unix.go`: Unix용 (`Setpgid: true`)
- `serve_windows.go`: Windows용 (`CREATE_NEW_PROCESS_GROUP`)

## 변경 사항
- `internal/cli/serve_unix.go` 추가
- `internal/cli/serve_windows.go` 추가
- `internal/cli/serve.go`에서 직접 설정 대신 `setSysProcAttr()` 함수 호출

## 테스트
- [x] Linux amd64/arm64 빌드 성공
- [x] macOS amd64/arm64 빌드 성공
- [x] Windows amd64/arm64 빌드 성공
- [x] 기존 테스트 통과
