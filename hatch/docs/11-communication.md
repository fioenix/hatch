# 11 — Communication layer (agents nói chuyện trực tiếp)

Một đội người không chỉ để lại artifact — họ **nói chuyện**: hỏi nhau, @mention, họp. Đây là tầng đưa Hatch tới gần "human simulation" hơn các orchestrator thuần-task (vd Paperclip điều phối *qua* server trung tâm, **không** có giao tiếp trực tiếp giữa agent).

## Nguyên tắc: trực tiếp nhưng on-the-record

Agent là process cô lập — không socket thẳng vào nhau được, và **không nên** (mất audit, thành kênh ngầm). Nên giao tiếp "trực tiếp" ở Hatch nghĩa là: **có địa chỉ, theo lượt, qua một phương tiện chung** — đúng như con người cần Slack/phòng họp. Phương tiện đó là **bus + orchestrator**; mọi lượt nói đều được ghi lại.

| | **Bus** (`.hatch/bus/`) | **Ledger** (`.hatch/ledger/`) | **Board/KB** |
|---|---|---|---|
| Chở gì | *đối thoại* (hỏi/đáp/họp) | *sự kiện* trạng thái | *trạng thái* + *tri thức* |
| Hình thức | thread append-only | entry append-only | file ticket / KB |

> Board/ledger/KB vẫn là **nguồn sự thật cho trạng thái**. Bus chở **dialogue**. Quyết định chốt trong họp ⇒ ghi KB (ADR); đổi trạng thái ⇒ ledger.

## Mô hình Slack

| Slack | Hatch |
|---|---|
| Channel `#design` | conversation id `#design` (một file) |
| DM | conversation id `dm-codex-claude` (hoặc tự đặt) |
| Per-topic | dùng id ticket `T-123` làm channel |
| Thread (reply dưới một message) | message có `re:<rootId>` (cờ `--reply-to`) |
| Mở thread mới | post message mới (không `--reply-to`) |
| @mention | `@agent`/`@role` trong body → tự vào inbox người được tag |

## Cấu trúc

```
.hatch/bus/
├── threads/<channel>.md   # mỗi file = một CHANNEL/DM/conversation
└── .cursors.json          # con trỏ "đã đọc" của từng agent (tính inbox)
```

Mỗi message là một block append-only; reply tạo **thread** trong channel:
```
## <ts> · <from> → <to,...> · <type>[ · re:<rootId>] · {#<id>}
<body, có thể chứa @mention>
```
`type`: `msg` · `ask` (chờ trả lời) · `reply` · `decision` (chốt/đồng thuận).
`to`: id agent · id vai · `#channel` · `*`/`all`. **@mention trong body** tự được gộp vào `to` → người/vai được tag thấy trong `hatch inbox`.

## Ba kiểu giao tiếp

### 1. Channel · DM · @mention · thread (async)
```bash
# post vào channel, tag đồng đội ngay trong body
hatch msg --from codex -c '#design' "@claude-code @reviewer streaming hay buffer?"
# DM: channel riêng giữa hai agent
hatch msg --from codex -c dm-codex-claude "ping riêng nhé"
# reply trong thread (rooted tại message gốc) hoặc mở thread mới (không --reply-to)
hatch msg --from claude-code -c '#design' --reply-to m0614-145222-581011 "Streaming."
hatch inbox claude-code --mark     # DM + @mention + broadcast gửi tới mình; --mark = đã đọc
hatch channel ls                   # liệt kê channel/DM/conversation
hatch channel show '#design'              # cả channel
hatch channel show '#design' --in <root>  # chỉ một thread
```
Inbox = **notification kiểu Slack**: chỉ DM + @mention (id/vai) + broadcast `*`, không phải mọi message trong mọi channel. Channel thì *mở ra xem* bằng `hatch channel show`. Không chặn — người nhận xử lý ở lượt sau.

### 2. Hỏi-đáp đồng bộ (orchestrator relay)
```bash
hatch ask --from codex --to claude-code --thread T-123 "Chốt giúp: dùng streaming chứ?"
```
Orchestrator dựng prompt = bối cảnh thread + câu hỏi, **spawn agent đích headless**, bắt câu trả lời, ghi `reply` vào thread, trả về cho người hỏi. Đây là "nói chuyện" thật giữa hai agent — qua một phương tiện có ghi âm. `--dry-run` in invocation mà không chạy.

### 3. Họp nhiều agent (convene)
```bash
hatch convene --topic "Thiết kế export API" --agents claude-code,codex,kiro --rounds 2
```
Orchestrator chạy vòng luân phiên có giới hạn: mỗi vòng, mỗi agent thấy diễn biến thread tới giờ và đóng góp lượt của mình theo **vai** của nó. Agent mở đầu bằng `DECISION:` để chốt (message thành type `decision`). Đây là mô phỏng một cuộc họp/đánh giá thiết kế đầy đủ — kết quả nằm trong thread, sẵn sàng đề bạt sang KB.

## Vì sao mạnh hơn task-only orchestration
- **Hỏi để gỡ vướng ngay** thay vì block ticket chờ vòng sau.
- **Tranh luận thiết kế** (convene) cho ra quyết định tốt hơn một agent đơn lẻ.
- **Vẫn audit đầy đủ**: mọi câu đều có `from/to/ts/thread`, tái lập được "ai nói gì, vì sao".
- **Không khóa cứng đồng bộ**: DM async cho luồng rời; ask/convene cho lúc cần đối thoại.

## Giới hạn hiện tại
- Relay là **bán đồng bộ**: orchestrator chạy agent đích một lượt rồi lấy stdout làm câu trả lời (chưa phải phiên chat dài nhiều lượt giữ session). Mở rộng: dùng `--resume` của từng agent để giữ mạch hội thoại.
- Body chứa heading `## ` có thể làm parser tách nhầm — tránh dùng `## ` đầu dòng trong message (sẽ được xử lý ở bản sau).
