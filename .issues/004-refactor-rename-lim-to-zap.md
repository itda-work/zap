---
number: 4
title: "refactor: 프로젝트명 lim → zap 변경"
state: done
labels:
  - refactor
assignees:
  - allieus
created_at: 2026-01-15T18:39:24Z
updated_at: 2026-01-16T23:29:56Z
closed_at: 2026-01-15T00:00:00Z
---

## 개요

프로젝트명과 CLI 명령어를 `lim`에서 `zap`으로 변경합니다.

## 배경

- 프로젝트 디렉토리가 `zap`으로 변경됨
- GitHub 저장소가 `itda-work/zap`으로 설정됨
- 일관성을 위해 명령어도 `zap`으로 변경 필요

## 변경 범위

### 1. Go 모듈명 변경
- `github.com/allieus/lim` → `github.com/itda-work/zap`
- `go.mod` 수정
- 모든 import 경로 업데이트

### 2. 디렉토리 구조 변경
- `cmd/lim/` → `cmd/zap/`

### 3. 소스 코드 내 참조 변경
- CLI help 텍스트, 명령어 예시
- README.md 문서

### 4. Git remote 설정
- `git@github.com:itda-work/zap.git` remote 추가

### 5. GitHub Actions 릴리즈 워크플로우
- version tag push 시 자동 릴리즈 빌드
- 멀티 플랫폼 바이너리 생성 (Linux, macOS, Windows)

## 작업 목록

- [x] Git remote 추가
- [x] go.mod 모듈명 변경
- [x] cmd/lim → cmd/zap 디렉토리 이동
- [x] 소스 코드 내 lim → zap 참조 변경
- [x] README.md 업데이트
- [x] Makefile 업데이트
- [x] GitHub Actions 릴리즈 워크플로우 추가
- [x] 빌드 및 테스트 검증
- [x] v0.2.0 릴리즈

## 진행 내역

### 2026-01-15

#### 구현 완료

1. **Git remote 추가**: `origin` → `git@github.com:itda-work/zap.git`

2. **모듈명 변경**: `github.com/itda-work/zap`

3. **디렉토리 이동**: `cmd/lim/` → `cmd/zap/`

4. **소스 코드 참조 변경**:
   - `cmd/zap/main.go`: import 경로
   - `internal/cli/*.go`: import 경로, CLI help 텍스트
   - `internal/cli/init.go`: 생성되는 지침 파일 내 명령어 예시

5. **문서 업데이트**:
   - `README.md`: 설치/사용법 명령어 변경
   - `Makefile`: 빌드 타겟 및 ldflags 변경

6. **GitHub Actions 릴리즈 워크플로우**:
   - `.github/workflows/release.yml` 생성
   - `v*` 태그 push 시 자동 실행
   - 멀티 플랫폼 바이너리 빌드
   - 자동 릴리즈 생성 및 체크섬 파일 포함

7. **Makefile 개선**:
   - `make build-all`: 모든 플랫폼 빌드 (dist/ 폴더)
   - `make release TAG=vX.Y.Z`: 빌드 + GitHub 릴리즈 생성
   - 지원 플랫폼: Linux/macOS/Windows × amd64/arm64

8. **버전 출력 개선**:
   - 빌드 날짜 포함: `zap version v0.2.0 (built 2026-01-15)`
   - ldflags로 Version, BuildDate 주입

9. **v0.2.0 릴리즈 완료**:
   - https://github.com/itda-work/zap/releases/tag/v0.2.0
   - 6개 바이너리 + checksums.txt
