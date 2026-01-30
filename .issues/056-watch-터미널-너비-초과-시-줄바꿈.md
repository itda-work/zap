---
number: 56
title: 'watch: 터미널 너비 초과 시 줄바꿈 대신 1줄 내 truncate'
state: done
labels:
    - cli,watch
assignees: []
created_at: "2026-01-30T05:42:54Z"
updated_at: "2026-01-30T05:44:55Z"
closed_at: "2026-01-30T05:44:55Z"
---

zap watch에서 터미널 width가 좁아지면 내용이 다음 줄로 넘어감. ANSI/CJK-aware truncate 적용하여 1줄 내에서만 표시되도록 수정.
