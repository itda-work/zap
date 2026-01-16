---
title: "feat: zap update 자동 업데이트 명령"
state: done
labels:
  - feature
  - cli
assignees: []
created_at: 2026-01-16
updated_at: 2026-01-16
---

## 설명

`zap update` 명령을 통해 새 버전 확인 및 자동 업데이트 기능 구현

## 구현 내용

### 명령어
```bash
zap update              # 새 버전 확인 후 업데이트 (확인 프롬프트)
zap update --check      # 새 버전 확인만 (업데이트 안함)
zap update --force      # 확인 없이 바로 업데이트
zap update -v v0.3.0    # 특정 버전으로 업데이트
```

### 기능
- GitHub Releases API를 통한 최신 버전 확인
- Self-update: 바이너리가 스스로 새 버전을 다운로드하여 교체
- 체크섬 검증으로 다운로드 무결성 보장
- 원자적 바이너리 교체 (백업 → 교체 → 정리)
- Stable 릴리스만 지원 (pre-release 제외)

### 에러 처리
- 네트워크 오류 시 적절한 안내 메시지
- Rate limit 초과 시 대기 안내
- 권한 부족 시 sudo 사용 안내
- dev 빌드에서 업데이트 불가 안내

### 파일 구조
```
internal/
├── cli/
│   └── update.go           # Cobra 명령 정의
└── updater/
    ├── updater.go          # 업데이트 핵심 로직
    ├── github.go           # GitHub API 클라이언트
    ├── version.go          # 버전 비교 유틸리티
    └── version_test.go     # 테스트
```

## 완료 기준

- [x] `zap update --check`로 새 버전 확인
- [x] `zap update`로 실제 업데이트 수행
- [x] 체크섬 검증
- [x] 권한 오류 처리
- [x] dev 빌드 감지 및 안내
- [x] 단위 테스트 작성
