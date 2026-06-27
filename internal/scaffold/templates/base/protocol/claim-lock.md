# Claim & Lock

- **Claim** = set `assignee` + `claim` trong frontmatter + `git mv` ticket vào `in-progress/`, rồi **push ngay**.
- **Push thắng = lock thắng.** Nếu push bị từ chối (người khác claim trước), rebase và chọn ticket khác.
- Không claim ticket còn `depends_on` chưa `done`.
- Tôn trọng WIP limit của agent (registry `wip`).
