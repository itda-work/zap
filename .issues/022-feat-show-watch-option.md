---
number: 22
title: 'feat(cli): show 명령에 --watch 및 --notify 옵션 추가'
state: done
labels:
    - enhancement
    - cli
assignees: []
created_at: 2026-01-17T15:00:00Z
updated_at: 2026-01-17T15:30:00+09:00
closed_at: 2026-01-17T15:30:00+09:00
---

## 개요

`zap show` 명령에 `tail -f`와 유사한 실시간 모니터링 옵션을 추가하여 이슈 파일 변경 시 즉시 갱신 출력하고, 상태가 done으로 변경되면 알림을 제공한다.

## 핵심 요구사항

| 항목 | 설명 |
|------|------|
| 파일 감시 | `fsnotify` 라이브러리로 이벤트 기반 감시 |
| 출력 방식 | 화면 초기화 후 전체 재렌더링 |
| 알림 기본 | Terminal Bell + 시각적 강조 (녹색 배너) |
| 시스템 알림 | `--notify` 플래그로 macOS 알림 센터 연동 |
| 알림 조건 | done 상태로 변경 시에만 |

## 명령어 인터페이스

### `zap show <number> -w`

이슈 파일 변경 시 실시간 갱신:

```bash
zap show 1 -w          # 기본 watch 모드
zap show 1 -w --raw    # raw 모드로 감시
zap show 1 -w --refs   # refs 그래프 포함 감시
```

### `zap show <number> -w --notify`

done 상태 변경 시 macOS 시스템 알림 추가:

```bash
zap show 1 -w --notify
```

## 구현 내용

### 변경 파일

- `go.mod`: `github.com/fsnotify/fsnotify v1.9.0` 의존성 추가
- `internal/cli/show.go`:
  - `-w, --watch` 플래그
  - `--notify` 플래그
  - `watchIssue()`: fsnotify 기반 파일 감시 루프
  - `notifyDone()`: 벨 + 시각적 알림
  - `sendSystemNotification()`: macOS osascript 연동

### 알림 표시 예시

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ Issue #1 marked as done!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## 기술 결정

| 항목 | 결정 | 이유 |
|------|------|------|
| 파일 감시 | fsnotify | 이벤트 기반, CPU 효율적, 즉각 반응 |
| 출력 방식 | 전체 재렌더링 | frontmatter 변경도 반영 필요 |
| 디바운싱 | 50ms | 에디터 저장 패턴(임시파일→이름변경) 대응 |
