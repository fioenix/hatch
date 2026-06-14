# 06 — Governance & Audit

Nhiều agent tự động chạm vào code → rủi ro tăng. Hatch đặt ba lớp kiểm soát: **audit trail đầy đủ**, **gates trước hành động không thể hoàn tác**, và **giới hạn thẩm quyền agent**. Khung này hợp với môi trường doanh nghiệp cần truy vết (vd IPO readiness, compliance).

## Ledger — sổ audit append-only

Mỗi thay đổi trạng thái có ý nghĩa sinh một entry. Ledger là **append-only** (không sửa/xóa entry cũ — sửa = thêm entry đính chính), nằm trong git nên bất biến kép.

Một entry tối thiểu trả lời 5 câu: **WHO · WHAT · WHEN · WHY · WHERE**.

```
## 2026-06-14T09:12:03+07:00 · codex · T-123
- action: claim
- from: backlog/ → in-progress/
- why: sprint S-7, ưu tiên cao, dependency T-100 đã done
- branch: hatch/T-123-export-csv
```

Xem schema đầy đủ: [spec/ledger](../spec/ledger.schema.md).

Ledger phục vụ: standup digest, retro, điều tra sự cố, và **chứng cứ audit** "agent nào làm gì, vì sao, lúc nào".

## Gates — chặn trước hành động rủi ro

Gate = điều kiện bắt buộc pass trước khi qua một ranh giới. Hai loại:

### Gate tự động (agent/CI tự chạy)
- **DoD gate** (Reviewer): checklist [protocol/definition-of-done](03-coordination-protocol.md#definition-of-done-dod).
- **Test/lint gate:** xanh mới được vào review/.
- **No-self-review:** Implementer ≠ Reviewer trên cùng ticket (registry enforce).
- **Stale-context gate:** SSOT đổi mà chưa compile → chặn (xem [compiler](04-context-compiler.md#stale-detection)).

### Human gate (bắt buộc người duyệt)
Một số hành động **agent không bao giờ tự làm** — luôn dừng và chờ người:

| Hành động | Vì sao cần người |
|---|---|
| Merge vào nhánh chính | Quyết định cuối cùng thuộc về người |
| Deploy production | Không thể hoàn tác, ảnh hưởng thật |
| Xóa data / migration phá hủy | Mất mát không hồi |
| Đụng secret / credential | Bảo mật |
| Gửi communication ra ngoài | Brand/pháp lý |
| Quyết định pháp lý/tài chính ràng buộc | Ngoài thẩm quyền agent |

Khai báo trong registry: `policy.require-human-gate: [merge, deploy, secret, external-comms, destructive-data]`.

## Giới hạn thẩm quyền theo vai

Ranh giới mỗi vai (cột "KHÔNG được" ở [roles](02-roles.md)) là một dạng governance: Implementer không đổi scope, Reviewer không tự sửa rồi tự duyệt, v.v. Vi phạm ranh giới = ticket bị Reviewer trả về.

## Branch & PR là biên giới an toàn

- **Branch-per-ticket:** mọi thay đổi cô lập trên nhánh `hatch/T-xxx`. Nhánh chính luôn sạch.
- **PR bắt buộc để merge:** PR là nơi human gate diễn ra — review cuối + CI + phê duyệt. Agent mở PR (draft), người duyệt và merge.
- Không agent nào push thẳng nhánh chính.

## Sự cố & leo thang

- Agent phát hiện rủi ro vượt thẩm quyền → set `blocked`, ghi ledger, **không tự xử**.
- Conductor quét blocker; cái nào thuộc human gate → leo thang cho người (không tự gỡ).
- Sự cố nghiêm trọng (lộ secret, phá data) → dừng ngay, ghi ledger mức cao nhất, báo người.

## Nguyên tắc trung thực (áp cho mọi agent)

Mượn thẳng từ chuẩn làm việc nghiêm túc:
- Báo cáo kết quả đúng sự thật: test fail thì nói fail kèm output; bước bị skip thì nói skip.
- Không bịa số liệu/nguồn. Phân biệt dữ kiện / suy luận / giả định.
- Việc xong và đã verify mới được tuyên bố "done".

Những nguyên tắc này nên nằm trong `charter.md` (L0) để mọi agent đều nạp.
