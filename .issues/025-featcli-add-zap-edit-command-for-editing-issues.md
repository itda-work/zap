---
number: 25
title: 'feat(cli): add zap edit command for editing issues'
state: done
labels:
    - enhancement
    - cli
assignees: []
created_at: 2026-01-18T10:13:20.805589+09:00
updated_at: 2026-01-18T10:18:31.081121+09:00
closed_at: 2026-01-18T10:18:31.081121+09:00
---

### 개요

이슈 파일을 에디터로 열어 직접 편집할 수 있는 `zap edit` 명령을 추가합니다.

### 상세 요구사항

#### 기본 사용법

```bash
zap edit <number>    # 이슈 #N의 마크다운 파일을 에디터로 열기
```

#### 에디터 선택 우선순위 (Git 방식)

Git commit과 동일한 방식으로 에디터를 결정합니다:

1. `GIT_EDITOR` 환경변수
2. `VISUAL` 환경변수
3. `EDITOR` 환경변수
4. 기본값:
   - Unix/macOS: `vi`
   - Windows: `notepad`

#### 동작 흐름

1. 이슈 번호로 파일 경로 확인 (예: `.issues/001-feat-example.md`)
2. 이슈 파일이 존재하지 않으면 에러 출력
3. 환경변수 순서대로 에디터 탐색
4. 에디터 실행 (blocking - 사용자가 저장 후 종료할 때까지 대기)
5. 에디터 종료 후 명령 완료

#### 에디터 옵션

- 별도의 `-e`, `--editor` 옵션은 지원하지 않음
- 사용자가 에디터를 변경하려면 환경변수를 설정해야 함

```bash
# 예시: VS Code로 편집하려면
EDITOR='code --wait' zap edit 1

# 또는 환경변수 영구 설정
export EDITOR='code --wait'
zap edit 1
```

### 구현 참고사항

- Rust의 `std::env::var()` 로 환경변수 읽기
- `std::process::Command` 로 에디터 실행
- 에디터는 foreground에서 실행되어야 함 (stdin/stdout 연결)
- 에디터 종료 코드 확인 (0이 아니면 경고)

### 관련 자료

- [Git Editor 설정 문서](https://git-scm.com/book/en/v2/Customizing-Git-Git-Configuration#_core_editor)
