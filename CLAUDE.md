# Zap

## zap - Local Issue Management

### 중요: GitHub 이슈가 아닌 로컬 이슈 사용

이 프로젝트는 로컬 이슈 관리 시스템(.issues/)을 사용합니다.
이슈 관리 시 `gh issue` 명령이 아닌 `zap` 명령을 사용하세요. 올바른 형식이 자동으로 적용됩니다.

### zap CLI 명령어

#### 명령 예시

```bash
zap new "제목"              # 새 이슈 생성 (PDCA 템플릿 자동 포함)
zap new "제목" -l label     # 레이블과 함께 생성
zap new "제목" -a user      # 담당자와 함께 생성
zap new "제목" -b "본문"    # 본문과 함께 생성

zap list                    # 목록 조회
zap show 1                  # 상세보기

# 상태 변경 (PDCA 워크플로우)
# 상태 변경 시 파일의 frontmatter가 업데이트됩니다 (파일 위치 변경 없음):
zap set open 1              # state: open (이슈 재오픈)
zap set wip 1               # state: wip (작업 시작 - Do)
zap set check 1             # state: check (자기 검증 - Check)
zap set review 1            # state: review (외부 리뷰 - Review)
zap set done 1              # state: done (작업 완료 - Act)
zap set closed 1            # state: closed (취소/보류)
```

**PDCA 상태 모델:**

| 상태 | PDCA | 의미 |
|------|------|------|
| `open` | Plan | 계획 수립, 작업 대기 |
| `wip` | Do | 구현 진행 중 |
| `check` | Check | 자기 검증 (Plan 기준 대비) |
| `review` | Review | 외부 리뷰, 피드백 수집 |
| `done` | Act | 완료, 개선 조치 기록 |
| `closed` | - | 취소/보류 (어느 단계에서든) |

**done vs closed 핵심 구분:**
- 코드를 작성/수정했다 → `done`
- 작업 없이 닫는다 → `closed`

**갭 발견 시 반복:** check/review에서 갭이 발견되면 `zap set wip <number>`로 되돌려 재작업할 수 있습니다.

### 커밋 메시지 규칙

이슈 관련 커밋은 메시지 끝에 이슈 번호를 포함하고, 구현이 완료되면 구현 내역과 이슈 파일을 함께 커밋하세요:

```bash
git commit -m "feat: 로그인 기능 구현 (#1)"
git commit -m "fix: 버그 수정 (#23)"
```

**Skills/Commands/Agents 사용 기록 (선택사항):**

이슈 작업 중 skills, commands, agents를 사용한 경우, 커밋 메시지 하단(footer)에 기록하면 작업 컨텍스트를 유지하는 데 도움이 됩니다:

```bash
git commit -m "feat: 게임 초기화 구조 구현 (#5)

GDevelop 기반의 프로젝트 구조 설정 및 기본 씬 구성 완료

Skills: /game:init (프로젝트 템플릿 생성)
Commands: /clarify (요구사항 명확화)"
```

포함할 정보:
- `Skills:` - 사용한 skill과 목적 (예: `/game:init (프로젝트 템플릿)`)
- `Commands:` - 사용한 command와 목적 (예: `/clarify (요구사항 정리)`)
- `Agents:` - 사용한 agent와 목적 (예: `codex-exec (알고리즘 최적화)`)

### 워크플로우 (PDCA)

1. **이슈 생성 (Plan)**: `zap new "이슈 제목"`
2. **작업 시작 (Do)**: `zap set wip <number>`
3. **자기 검증 (Check)**: `zap set check <number>`
4. **외부 리뷰 (Review)**: `zap set review <number>`
5. **작업 완료 (Act)**: `zap set done <number>`
6. **취소/보류**: `zap set closed <number>`

### 주의사항

- **이슈 생성 시 반드시 `zap new` 명령을 사용하세요** (파싱 오류 방지)
- 이슈 번호는 고유해야 합니다
- 파일명의 번호와 frontmatter의 number가 일치해야 합니다
- 상태 변경 시 frontmatter의 state 필드가 업데이트됩니다
