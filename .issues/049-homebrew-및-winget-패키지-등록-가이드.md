---
number: 49
title: Homebrew 및 winget 패키지 등록 가이드 문서 작성
state: done
labels:
    - docs
assignees: []
created_at: "2026-01-24T13:55:11Z"
updated_at: "2026-01-24T13:58:42Z"
closed_at: "2026-01-24T13:58:42Z"
---

## 개요

zap CLI를 Homebrew(macOS/Linux)와 winget(Windows)에 등록하기 위한 실제 단계별 가이드 문서 작성

## 결과물

`docs/PACKAGING.md`

## 문서 구성

### 1. Homebrew 등록
- 1.1 자체 tap 생성 및 운영
  - homebrew-tap 저장소 생성
  - Formula 작성법 (zap.rb)
  - GitHub Actions 자동화
  - 사용자 설치 방법
- 1.2 homebrew-core 공식 등록
  - 등록 조건 (notable, 75+ stars 등)
  - Formula 작성 가이드라인
  - PR 제출 절차
  - 리뷰 대응

### 2. winget 등록
- 2.1 자체 manifest 저장소 운영
  - manifest 파일 구조
  - 로컬 테스트 방법
  - 사용자 설치 방법
- 2.2 winget-pkgs 공식 등록
  - manifest 작성 (YAML 3파일 구조)
  - wingetcreate 도구 활용
  - PR 제출 절차
  - 자동 업데이트 설정 (komac)

### 3. 릴리즈 자동화 연동
- GitHub Actions에서 자동 Formula/manifest 업데이트
- 체크섬 및 버전 관리
