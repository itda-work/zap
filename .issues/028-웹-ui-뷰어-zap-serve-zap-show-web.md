---
number: 28
title: 'feat: 웹 UI 뷰어 (zap serve, zap show --web)'
state: done
labels:
    - enhancement
    - feature
assignees: []
created_at: 2026-01-18T12:07:40.243085+09:00
updated_at: 2026-01-18T18:56:17.619306+09:00
closed_at: 2026-01-18T18:56:17.619306+09:00
---

## 개요

터미널에서 `zap show` 출력의 가독성 한계를 해결하기 위해, 웹 브라우저 기반 이슈 뷰어를 제공합니다.

## 핵심 기능

### 1. `zap serve`
- 전체 이슈 대시보드를 웹 UI로 제공
- 이슈 목록 + 상세 보기 통합 인터페이스
- 라이브 리로드: 이슈 파일 변경 시 자동 새로고침

### 2. `zap show <number> --web`
- 단일 이슈를 웹 브라우저로 열기
- 해당 이슈만 표시하는 단일 페이지 뷰
- 라이브 리로드 지원

## 기술 스펙

### 서버
- Go 내장 `net/http` 서버
- 파일 시스템 워치 (fsnotify 등)
- WebSocket 또는 SSE를 통한 라이브 리로드
- 시스템 기본 브라우저 자동 열기

### 마크다운 렌더링
- 표 (tables)
- 코드 블록 (syntax highlighting)
- 이미지
- 링크
- 체크리스트

### 스타일링
- Tailwind CSS
- 다크 모드 지원 (시스템 설정 감지 + 수동 토글)
- 반응형 레이아웃

## 사용 시나리오

- 긴 이슈 내용 검토 시 스크롤/네비게이션 편의
- 팀원과 화면 공유 시 시각적 품질
- 마크다운 렌더링 결과 확인
- 코딩 중 별도 창에 이슈 띄워놓고 참조

## 데몬 모드 (백그라운드 실행)

### 명령 구조

```bash
zap serve              # 포그라운드 실행 (기본)
zap serve -D           # 백그라운드(데몬) 실행
zap serve stop         # 데몬 중지
zap serve status       # 데몬 상태 확인 (실행 여부, 포트, PID)
zap serve logs         # 로그 확인 (tail -f)
```

### 구현 방식

- PID 파일: `.issues/.zap-serve.pid`
- 로그 파일: `.issues/.zap-serve.log`
- `-d` 실행 시 `zap serve` 프로세스를 detach하여 백그라운드 실행
- `stop`은 PID 파일을 읽어 SIGTERM 전송
- `status`는 PID 파일 확인 + 프로세스 생존 여부 체크

### 플래그

| 플래그 | 설명 |
|--------|------|
| `-D, --daemon` | 백그라운드(데몬) 모드로 실행 |
| `-p, --port` | 포트 지정 (기본: 18080) |
| `--no-browser` | 브라우저 자동 열기 비활성화 |

## 구현 완료

### 생성된 파일

```
internal/web/
├── markdown.go      # goldmark 기반 마크다운 → HTML 렌더링 (syntax highlighting 포함)
├── templates.go     # embed.FS 기반 HTML 템플릿 + 헬퍼 함수
├── handlers.go      # HTTP 핸들러 (Dashboard, ListIssues, GetIssue, ViewIssue)
├── server.go        # HTTP 서버 + SSE + 파일 워치 + Access 로깅
├── daemon.go        # PID/로그 파일 관리, 데몬 상태 체크
└── templates/
    ├── dashboard.html   # 대시보드 페이지
    └── issue.html       # 이슈 상세 페이지

internal/cli/
├── serve.go         # zap serve 명령 (stop, status, logs 서브커맨드 포함)
└── show.go          # --web 플래그 추가
```

### 기술 스택

| 구분 | 기술 |
|------|------|
| HTTP 서버 | Go 내장 `net/http` |
| 라이브 리로드 | SSE (Server-Sent Events) |
| 파일 워치 | `fsnotify` (기존 의존성) |
| 마크다운 | `goldmark` + `goldmark-highlighting` |
| 구문 강조 | `chroma` (기존 의존성) |
| CSS | Tailwind CSS (CDN) |
| 브라우저 열기 | `os/exec` + 플랫폼별 명령 |

### API 엔드포인트

| 경로 | 설명 |
|------|------|
| `GET /` | 대시보드 HTML |
| `GET /issues` | 이슈 목록 JSON |
| `GET /issues/:number` | 단일 이슈 JSON (HTML body 포함) |
| `GET /issues/:number/view` | 단일 이슈 HTML 페이지 |
| `GET /events` | SSE 엔드포인트 (라이브 리로드) |

### 서버 설정

- **기본 포트**: 18080
- **타임아웃**: ReadTimeout(15s), WriteTimeout(60s), IdleTimeout(60s)
- **SSE Heartbeat**: 30초 간격
- **Access 로깅**: 모든 요청에 대해 `[req#] METHOD PATH STATUS DURATION` 형식 출력

### SSE 연결 관리

- `beforeunload` 이벤트로 페이지 이동 시 연결 정리
- `visibilitychange` 이벤트로 탭 전환 시 연결 관리
- 연결/해제 시 클라이언트 수 로깅

## 참고

- GitHub Issues 스타일과 유사한 룩앤필 지향
- 최소 의존성으로 Go 바이너리에 포함
