---
number: 45
title: 'feat(cli): light 테마 지원 - OSC 11 기반 터미널 테마 감지'
state: done
labels:
    - enhancement
    - cli
assignees: []
created_at: "2026-01-23T03:27:15Z"
updated_at: "2026-01-23T03:29:18Z"
closed_at: "2026-01-23T03:29:18Z"
---

## 개요
터미널 light 테마에서도 색상이 잘 보이도록 테마별 색상 지원 추가

## 구현 방식
1. OSC 11 쿼리로 터미널 배경색 감지 (타임아웃 100ms)
2. ZAP_THEME 환경변수로 override 가능
3. 감지 실패 시 dark 기본값

## 색상 전략
- Light 테마: 어두운 색상 사용 (진한 녹색, 진한 회색 등)
- Dark 테마: 기존 밝은 색상 유지

## 구현 내역

### 변경 파일
- `internal/cli/color.go`: 테마 감지 및 테마별 색상 시스템 추가

### 테마 감지 우선순위
1. `ZAP_THEME` 환경변수 (light/dark)
2. OSC 11 쿼리 (터미널 배경색 직접 조회, 100ms 타임아웃)
3. `COLORFGBG` 환경변수 (일부 터미널 지원)
4. 기본값: dark

### Light 테마 색상 매핑
| 용도 | Dark 테마 | Light 테마 |
|------|-----------|------------|
| 밝은 녹색 | `\033[92m` | `\033[38;5;22m` (진한 녹색) |
| 밝은 노랑 | `\033[93m` | `\033[38;5;136m` (올리브) |
| 회색 | `\033[90m` | `\033[38;5;240m` (진한 회색) |
| 배경 | `\033[48;5;238m` | `\033[48;5;253m` (밝은 회색) |

### 사용법
```bash
# 자동 감지 (기본)
zap list

# Light 테마 강제
ZAP_THEME=light zap list

# Dark 테마 강제
ZAP_THEME=dark zap list
```
