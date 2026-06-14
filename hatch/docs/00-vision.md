# 00 — Vision

## Vấn đề

Một developer hôm nay không còn dùng một agent. Họ có Claude Code làm chính, thêm Codex, Kiro, Antigravity CLI cho những việc khác nhau. Nhưng khi nhiều agent cùng đụng vào **một repo**, ba thứ vỡ ra:

1. **Instructions phân mảnh & drift.** Mỗi agent đọc một file khác nhau (`CLAUDE.md`, `AGENTS.md`, `.kiro/steering/`…). Người dùng phải viết tay từng cái, rồi chúng lệch nhau theo thời gian. Cùng một "code style" được phát biểu 4 lần, 4 phiên bản hơi khác.

2. **Lãng phí token.** Không có ai phân tầng context. Hoặc mỗi agent đọc tất cả (phình context mỗi message), hoặc đọc thiếu (làm sai). Không có khái niệm "agent này chỉ cần biết phần này".

3. **Hỗn loạn khi cùng làm.** Hai agent sửa cùng một file. Không ai biết agent kia đang làm gì. Không có hàng đợi việc, không có hand-off, không có audit trail. Kết quả là conflict, việc trùng, và mất dấu vết "ai làm gì vì sao".

overclaud đã giải quyết vế (1) và (2) cho **một** agent (Claude) qua kiến trúc layering + token optimization. Hatch mở rộng cả ba vế cho **nhiều** agent.

## Tầm nhìn

> Một repo, nhiều agent, vận hành như một squad Agile gọn — có charter chung, vai rõ ràng, bảng việc, quy ước phối hợp, và sổ ghi đầy đủ — với chi phí token tối thiểu vì mỗi agent chỉ nạp đúng phần của mình.

Hatch là **harness**: bộ giàn giáo bao quanh để *ràng buộc* (mỗi agent biết vai, ranh giới, context của mình) và *điều phối* (các agent phối hợp qua artifact chung, không giẫm chân nhau).

## Nguyên tắc neo: mô phỏng một squad người

Đây không phải ẩn dụ trang trí — nó là **nguyên tắc thiết kế ràng buộc**. Mỗi khi thiết kế một cơ chế Hatch, ta hỏi: *"Một đội người làm việc này ra sao?"* và sao chép cấu trúc đó, vì:

- Quy trình squad người đã được tối ưu hàng chục năm cho đúng bài toán: nhiều tác nhân thông minh nhưng **không chia sẻ bộ nhớ**, phối hợp qua **artifact bên ngoài** (bảng việc, tài liệu, code review), bất đồng bộ.
- Coding agent cũng đúng đặc tính đó: thông minh, không chung memory, mỗi con một process. Nên giải pháp của loài người áp gần như 1:1.

Hệ quả thiết kế trực tiếp:
- Agent **không gọi trực tiếp** agent khác (người cũng không "ghi đè bộ nhớ" đồng nghiệp) — phối hợp qua board + ledger.
- Có **một người điều phối** lập kế hoạch (Conductor = EM/Scrum Master).
- Mỗi việc là một **ticket** có vòng đời rõ ràng.
- Mọi thay đổi trạng thái để lại **dấu vết** (ledger = standup notes + git log).
- Người mới vào được phát **handbook đúng vai** (compiler = onboarding).
- Tri thức tích lũy vào **wiki chung** (KB) — không đọc được não nhau, nhưng cùng tra/cập nhật một chỗ.
- **Vai trò và quy trình do đội tự định ở mỗi project** — không áp một khuôn cứng cho mọi nơi.

## Mục tiêu đo được

| Mục tiêu | Chỉ số |
|---|---|
| Hết drift instruction | 1 nguồn canonical, N output compiled tự động, 0 chỉnh tay file output |
| Tiết kiệm token | Mỗi agent nạp ≤ L0+L1+ticket; không nạp context ngoài vai |
| Không giẫm chân | 0 ticket bị 2 agent claim cùng lúc; branch-per-ticket |
| Audit đầy đủ | Mọi chuyển trạng thái ticket có entry ledger (who/what/when/why) |
| Onboard agent mới nhanh | Thêm 1 agent = 1 dòng `registry.yaml` + 1 lần compile |
| Tri thức không bốc hơi | Quyết định/bài học ghi vào KB; agent sau tra cứu thay vì dò lại |
| Linh hoạt per-project | Vai trò + workflow cấu hình riêng từng project, không sửa code |

## Ngoài phạm vi (bản đầu)

- Không tự huấn luyện/đánh giá model.
- Không thay thế CI/CD — Hatch *gọi* gate (test/lint/review) chứ không tự là CI.
- Không quản lý multi-repo (bản đầu giả định một repo; multi-repo là tương lai).
