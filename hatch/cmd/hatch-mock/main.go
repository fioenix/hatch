// Command hatch-mock is a stand-in coding agent for testing Hatch end-to-end
// without a real agent CLI installed. It reads a prompt (from --prompt or
// stdin) and prints a short, deterministic reply shaped by the prompt's role
// cues, so execute/relay/pair/convene flows can be exercised in CI/remote.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	prompt := flag.String("prompt", "", "task/prompt (else read from stdin)")
	flag.Parse()

	text := *prompt
	if text == "" {
		if b, _ := io.ReadAll(os.Stdin); len(b) > 0 {
			text = string(b)
		}
	}
	fmt.Println(reply(text))
}

// reply returns a deterministic mock response based on cues in the prompt.
func reply(p string) string {
	first := firstLine(p)
	switch {
	case strings.Contains(p, "NAVIGATOR"):
		return "READY (mock): scope khớp, test ổn — chuyển review được."
	case strings.Contains(p, "phân xử") || strings.Contains(p, "CHƯA chốt"):
		return "DECISION: (mock) chọn phương án đơn giản nhất, đủ đáp ứng yêu cầu."
	case strings.Contains(p, "DRIVER"):
		return "(mock) Đã thêm một bước nhỏ theo ticket; cần navigator soi phần lỗi biên."
	case strings.Contains(p, "vòng") || strings.Contains(p, "họp"):
		return "(mock) Tán thành hướng hiện tại; lưu ý cover ca biên."
	default:
		return "(mock) Đã xử lý: " + truncate(first, 80)
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
