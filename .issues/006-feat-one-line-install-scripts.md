---
number: 6
title: 'feat: One-Line 설치 스크립트 제공'
state: done
labels:
    - feature
    - docs
assignees:
    - allieus
created_at: 2026-01-16T08:59:13Z
updated_at: 2026-01-16T23:29:56Z
---

## 개요

GitHub 릴리즈에서 zap 바이너리를 OS별로 쉽게 설치할 수 있는 one-line 스크립트 제공

## 배경

현재 설치 방법:
- `go install` (Go 설치 필요)
- 소스에서 빌드

GitHub 릴리즈에 바이너리가 있지만, 수동 다운로드가 필요함

## One-Line 설치 명령

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/itda-work/zap/main/scripts/install.sh | bash
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/itda-work/zap/main/scripts/install.ps1 | iex
```

## 작업 목록

- [x] scripts/install.sh 생성
- [x] scripts/install.ps1 생성
- [x] README.md 업데이트

## 진행 내역

### 2026-01-16

- 이슈 생성
- 구현 완료:
  - `scripts/install.sh`: macOS/Linux용 설치 스크립트
  - `scripts/install.ps1`: Windows PowerShell용 설치 스크립트
  - README.md: One-line 설치 명령, closed 상태 문서 추가
