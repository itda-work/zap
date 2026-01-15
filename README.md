# zap - Local Issue Manager

`.issues/` 디렉토리 내 로컬 이슈를 관리하는 CLI 도구입니다.

## 설치

```bash
go install github.com/itda-work/zap/cmd/zap@latest
```

또는 소스에서 빌드:

```bash
git clone https://github.com/itda-work/zap.git
cd zap
make install
```

## 사용법

```bash
# 이슈 목록
zap list                    # 활성 이슈 (open + in-progress)
zap list --all              # 전체 이슈
zap list --state done       # 특정 상태
zap list --label bug        # 레이블 필터

# 이슈 상세
zap show 1                  # 이슈 #1 상세
zap show 1 --raw            # 원본 마크다운

# 상태 변경
zap open 1                  # → open/
zap start 1                 # → in-progress/
zap done 1                  # → done/

# 검색 & 통계
zap search "키워드"          # 제목/내용 검색
zap stats                   # 통계 대시보드

# AI 에이전트 지침 파일 생성
zap init claude             # CLAUDE.md 생성
zap init codex              # AGENTS.md 생성
zap init gemini             # GEMINI.md 생성
zap init claude --path AI_GUIDE.md  # 지정 파일에 생성
```

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
