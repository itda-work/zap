# Zap

## zap - Local Issue Management

### 중요: GitHub 이슈가 아닌 로컬 이슈 사용

이 프로젝트는 로컬 이슈 관리 시스템(.issues/)을 사용합니다.
이슈 관리 시 `gh issue` 명령이 아닌 `zap` 명령을 사용하세요. 올바른 형식이 자동으로 적용됩니다.

### zap CLI 명령어

#### 명령 예시

```bash
zap new "제목"              # 새 이슈 생성
zap new "제목" -l label     # 레이블과 함께 생성
zap new "제목" -a user      # 담당자와 함께 생성
zap new "제목" -b "본문"    # 본문과 함께 생성

zap list                    # 목록 조회
zap show 1                  # 상세보기

# 상태 변경
# 상태 변경 시 파일의 frontmatter가 업데이트됩니다 (파일 위치 변경 없음):
zap set open 1              # state: open (이슈 재오픈)
zap set wip 1               # state: wip (작업 시작)
zap set done 1              # state: done (작업 완료)
zap set closed 1            # state: closed (취소/보류)
```

**done vs closed 핵심 구분:**
- 코드를 작성/수정했다 → `done`
- 작업 없이 닫는다 → `closed`

### 커밋 메시지 규칙

이슈 관련 커밋은 메시지 끝에 이슈 번호를 포함하고, 구현이 완료되면 구현 내역과 이슈 파일을 함께 커밋하세요:

```bash
git commit -m "feat: 로그인 기능 구현 (#1)"
git commit -m "fix: 버그 수정 (#23)"
```

### 워크플로우

1. **새 이슈 생성**: `zap new "이슈 제목"` 실행
2. **작업 시작**: `zap set wip <number>` 실행
3. **작업 완료**: `zap set done <number>` 실행
4. **취소/보류**: `zap set closed <number>` 실행

### 주의사항

- **이슈 생성 시 반드시 `zap new` 명령을 사용하세요** (파싱 오류 방지)
- 이슈 번호는 고유해야 합니다
- 파일명의 번호와 frontmatter의 number가 일치해야 합니다
- 상태 변경 시 frontmatter의 state 필드가 업데이트됩니다
