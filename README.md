# lim - Local Issue Manager

`.issues/` 디렉토리 내 로컬 이슈를 관리하는 CLI/TUI 도구입니다.

## 설치

```bash
go install github.com/allieus/lim/cmd/lim@latest
```

또는 소스에서 빌드:

```bash
git clone https://github.com/allieus/lim.git
cd lim
make install
```

## 사용법

### CLI

```bash
# 이슈 목록
lim list                    # 활성 이슈 (open + in-progress)
lim list --all              # 전체 이슈
lim list --state done       # 특정 상태
lim list --label bug        # 레이블 필터

# 이슈 상세
lim show 1                  # 이슈 #1 상세
lim show 1 --raw            # 원본 마크다운

# 상태 변경
lim open 1                  # → open/
lim start 1                 # → in-progress/
lim done 1                  # → done/

# 검색 & 통계
lim search "키워드"          # 제목/내용 검색
lim stats                   # 통계 대시보드

# TUI 모드
lim                         # 인자 없으면 TUI
lim tui                     # 명시적 TUI
```

### TUI 키보드 단축키

| 키 | 동작 |
|---|------|
| `j/k` 또는 `↑/↓` | 이동 |
| `Enter` | 상세 보기 |
| `q` 또는 `Esc` | 뒤로/종료 |
| `1` | open만 표시 |
| `2` | in-progress만 표시 |
| `3` | done만 표시 |
| `0` | 전체 표시 |
| `/` | 검색 |
| `r` | 새로고침 |

## 이슈 파일 형식

`.issues/` 디렉토리 구조:

```
.issues/
├── open/           # 새로 생성된 이슈
├── in-progress/    # 진행 중
└── done/           # 완료
```

이슈 파일 (`001-feature-name.md`):

```markdown
---
number: 1
title: "이슈 제목"
state: open
labels:
  - feature
  - cli
assignees:
  - username
created_at: 2026-01-15T00:00:00Z
updated_at: 2026-01-15T00:00:00Z
closed_at:
---

## 개요

이슈 내용...

## 작업 목록

- [ ] 작업 1
- [x] 작업 2

## 진행 내역

### 2026-01-15

- 작업 내용
```

## 개발

```bash
# 의존성 설치
go mod download

# 빌드
make build

# 테스트
make test

# 로컬 설치
make install
```

## 라이선스

MIT
