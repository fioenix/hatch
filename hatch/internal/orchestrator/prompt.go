package orchestrator

import (
	"fmt"
	"strings"

	"github.com/fioenix/overclaud/hatch/internal/model"
)

// BuildPrompt produces the task prompt handed to an agent. The agent already
// loads L0+L1 from its compiled surface (CLAUDE.md/AGENTS.md/…); this prompt
// scopes it to one ticket and points at the L2 context to read on demand.
func BuildPrompt(t model.Ticket, role string) string {
	var b strings.Builder
	if role == "" {
		role = t.Role
	}
	fmt.Fprintf(&b, "Bạn đang làm việc với vai **%s** trên ticket **%s**: %s\n\n", role, t.ID, t.Title)
	b.WriteString("Quy trình bắt buộc:\n")
	fmt.Fprintf(&b, "1. Đọc ticket đầy đủ tại `.hatch/board/%s/%s` và mọi `context_refs` của nó.\n", t.Lane, t.Filename())
	b.WriteString("2. Tra `.hatch/kb/index.md` cho quyết định/bài học liên quan trước khi bắt tay.\n")
	b.WriteString("3. Thực thi đúng scope ticket, đạt Definition of Done (`.hatch/protocol/definition-of-done.md`).\n")
	b.WriteString("4. KHÔNG sửa file ngoài `context_refs` đã khai. Mở rộng scope ⇒ cập nhật ticket trước.\n")
	b.WriteString("5. Xong: cập nhật phần \"Handoff notes\" của ticket (đã làm gì / còn gì / cần gì).\n")
	b.WriteString("6. Tri thức đáng giữ ⇒ thêm vào `.hatch/kb/` (decision/learning/domain).\n\n")

	if len(t.ContextRefs) > 0 {
		b.WriteString("context_refs:\n")
		for _, r := range t.ContextRefs {
			fmt.Fprintf(&b, "  - .hatch/%s\n", r)
		}
		b.WriteString("\n")
	}
	if body := strings.TrimSpace(t.Body); body != "" {
		b.WriteString("--- Nội dung ticket ---\n")
		b.WriteString(body)
		b.WriteString("\n")
	}
	return b.String()
}

// BuildConsultPrompt frames a synchronous question from one agent to another:
// the recipient sees the thread so far and answers in character.
func BuildConsultPrompt(fromAgent, role, thread, threadRaw, question string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Bạn đang trả lời trực tiếp một đồng đội với vai **%s**.\n", role)
	fmt.Fprintf(&b, "Thread `%s`. %s hỏi bạn:\n\n%s\n\n", thread, fromAgent, question)
	if strings.TrimSpace(threadRaw) != "" {
		b.WriteString("--- Bối cảnh thread tới giờ ---\n")
		b.WriteString(strings.TrimSpace(threadRaw))
		b.WriteString("\n--- hết bối cảnh ---\n\n")
	}
	b.WriteString("Trả lời NGẮN GỌN, đúng trọng tâm, đúng vai. Chỉ in nội dung trả lời (sẽ được ghi vào thread).")
	return b.String()
}

// BuildMeetingPrompt frames one turn in a multi-agent meeting (convene).
func BuildMeetingPrompt(role, thread, topic, threadRaw string, round, rounds int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Bạn dự một cuộc họp đội với vai **%s** (vòng %d/%d).\n", role, round, rounds)
	fmt.Fprintf(&b, "Chủ đề: %s\n\n", topic)
	if strings.TrimSpace(threadRaw) != "" {
		b.WriteString("--- Diễn biến tới giờ ---\n")
		b.WriteString(strings.TrimSpace(threadRaw))
		b.WriteString("\n--- hết ---\n\n")
	}
	b.WriteString("Đóng góp lượt của bạn: phản hồi ý đồng đội, nêu lo ngại/đề xuất từ góc nhìn vai của bạn. ")
	b.WriteString("Nếu đã đồng thuận, mở đầu bằng `DECISION:` và chốt. In ngắn gọn (sẽ ghi vào thread).")
	return b.String()
}

// BuildPlanPrompt is the prompt for a Conductor planning pass.
func BuildPlanPrompt() string {
	return strings.Join([]string{
		"Bạn là **Conductor**. Lập kế hoạch cho chu kỳ tới.",
		"",
		"1. Đọc `.hatch/charter.md` (mission) và `.hatch/board/` (trạng thái hiện tại).",
		"2. Đọc `.hatch/kb/index.md` cho quyết định/ràng buộc liên quan.",
		"3. Bẻ epic/yêu cầu thành ticket nhỏ, mỗi ticket có `role`, `priority`, `depends_on`, acceptance rõ ràng.",
		"4. Tạo ticket bằng `hatch ticket new --title ... --role ... --priority ...` rồi điền nội dung.",
		"5. KHÔNG tự viết code production. KHÔNG tự merge.",
		"6. Ghi tóm tắt kế hoạch + lý do ưu tiên vào ledger (`why`).",
	}, "\n")
}
