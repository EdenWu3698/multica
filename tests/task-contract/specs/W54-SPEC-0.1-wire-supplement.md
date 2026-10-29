# W54-SPEC-0.1-wire-supplement

2026-10-29 · Samantha（需求）· WUSH-59 · **补充规格，提交 Asurada 核验；不是实现、正式回归或最终验收结论。**

只补 W54-SPEC-0.1 的 wire、准入绑定和测试接点。原 A-01–A-07、TC-01–TC-18、统一负向断言 N、权限及采用范围不变；以下是后续唯一实现契约需要表达的需求，不是另一份 canonical Schema。JARVIS 维护获授权实现契约；MOSS 维护正式测试；本次均不启动。

## 0. 实际输入与选择

| 输入 | 固定版本 / 核对结果 |
| --- | --- |
| WUSH-59 | revision 2；只交附件，不修改仓库或产品/测试代码 |
| WUSH-54 当前描述 | revision 20，2026-10-29T13:56:08Z；局部待接入，不整项停工；保留原验收标准 |
| 原规格附件 | `WUSH-54-task-context-contract-spec-v0.1.md`；附件 ID `01a18615-f19d-7717-8fb6-18027da920d2`；SHA256 `a3f3ba921b1f0edf8e371809c2e82be0149084dc6e3ace1fe906ba9f3c53791b` |
| MOSS 报告 | `WUSH-54-MOSS-result.md`；ID `01a18631-6f07-7bea-958e-7803d54c1713`；SHA256 `08e93239a02650ebab329c2913a0bfd255121b6c015f64b332c0ef77e05f6bec` |
| MOSS 证据包 | `WUSH-54-MOSS-evidence.tar.gz`；ID `01a18631-73e7-7e70-add2-34beac654a08`；SHA256 `82ffa8b23e9aebc9598e78fef348d4c9c2846956783be5e614acf5494dd9f5a6`；读取其中追溯、候选后端 builder、数据 manifest |
| 后续澄清 | 父任务根线程 `01a18631-753d-7a1b-aacc-eafe9db04f37`，展开至 MOSS `01a1879a-c0a0-78a1-9eef-ed4f5dca382e`（12:20:28Z）；环境已跑通，三组约定是局部依赖 |
| 测试成果 | `agent/moss/f8df4ba4dfb0`；base `8305976793097a3d38e3a2f29bc1626abe76130d`，报告实跑 head `d2038e096eff7cf4963e1ba7912efa6a4d3891cc`；87 通过 / 63 失败 / 0 跳过不是回归通过；本次不重复运行 |
| 本次只读 checkout | `https://github.com/EdenWu3698/multica`，`agent/samantha/a54251c01a68`，HEAD 为上述 base；无仓库修改；已读 `AGENTS.md` 及中英文 conventions |

本项目资源未提供 InFrame 开发主规范及 PRD/契约目录；不声称已读。MOSS 固定追溯明确本 Multica 任务已排除 InFrame 旧规范；本次不迁移旧工作区、不把 InFrame 规则带入本合同。

**收敛选择：** 保留 MOSS 候选的顶层完整 PUT、直接快照 GET、顶层 handoff/receipt 字段；`source_refs` 统一为内联完整 Source；复用既有 daemon 能力声明、认证、claim/start 和 `dispatched_at` fencing。不同方案（再套 `snapshot/data`、另建来源注册表、重建 runtime 协议）没有本批收益，不采用。对候选 builder 的必要改动仅为确定后的格式适配，不删断言、不重造业务校验器。

## 1. 通用 wire 与完整快照 PUT（WS-01）

### 1.1 公共规则

- 下文 `I` 表示 `/api/issues/{issue_id}/task-contract`，路由中的任务 ID 仍按现有资源解析；内部引用和 Actor ID 必须是 UUID。App API 身份及 `X-Workspace-ID` 沿用现有约束；不改 Public API v1、Plugin scopes 或 CLI 公共命令。
- JSON 为 UTF-8；键为 snake_case。请求拒绝重复键、未知字段、类型错误、非整数计数、非法 UUID。错误字段路径统一 **RFC6901 JSON Pointer**，根为 `""`；数组索引从 0 开始。响应可增加字段，客户端未知准入状态失败关闭，不回落 accepted。
- 除表中明确标为 nullable 的字段，`null` 一律非法；“允许空数组”不等于字段可省略。业务文字按去首尾 Unicode White_Space 后非空校验，存储及摘要保留原字节，不静默 trim；不接受占位词替代实际内容。
- 所有业务项 ID 是非空、区分大小写的局部标识，不要求 UUID；在整份快照中唯一。Actor 为 `{type:"member"|"agent",id:UUID}`，必须有效、同工作区且通过已有资源权限门；名字不是身份。
- `H(x)` 为 SHA256 原始字节的 64 位小写十六进制，无 `sha256:` 前缀。结构化摘要用 RFC8785 JCS 的 UTF-8、无 BOM/尾换行；只接受可规范化 JSON。摘要不是签名或授权凭证。

### 1.2 PUT 请求 / 成功响应

`PUT I` **没有 `snapshot` 或 `data` 包裹层**。请求 = 下表所有客户端字段 + `expected_version` + `expected_issue_revision`。不接收服务端生成字段，即使值看似正确也返回 400；Web 修改须从 GET 投影出可写字段，而非原样回传。

| 字段 | 请求约束（R=必填，O=可选） | 成功快照 |
| --- | --- | --- |
| `expected_version` | R，整数 ≥0；首次 0，后续为所基于合同版本 | 不进入快照 |
| `expected_issue_revision` | R，正整数，发布严格 CAS | 不进入快照 |
| `schema_version` | R，整数 1 | 原值 |
| `goal`, `goal_sources` | R，非空文本；Source[1..] | 原值 |
| `allowed_actions` | R，Action[1..]，见下 | 原值 |
| `prohibitions` | R，`{text,denied_actions}`；text 非空，DeniedAction[0..] | 原值 |
| `acceptance` | R，`{id,assertion,evidence_required,source_refs}`[1..]；后两者为非空字符串数组、Source[1..] | 原值 |
| `effective_inputs` | R，`{id,source,purpose,required}`[1..]；purpose 非空，required 为 boolean；包含主需求 | 原值 |
| `confirmed_facts` | R，`{id,assertion,source_refs}`[0..]；source_refs 非空 | 原值 |
| `assumptions` | R，`{id,assertion,validation,owner,blocks_start}`[0..]；validation 非空、owner Actor、blocks_start boolean | 原值 |
| `versions` | R，见下；不得提交 `issue_basis` | 补只读 `issue_basis` |
| `deliverables` | R，`{id,description,acceptance_ids,evidence_refs}`[1..]；acceptance_ids 非空且解析，evidence_refs 为 Source[0..] | 原值，不将空证据当已完成 |
| `open_questions` | R，`{id,question,owner,blocks_start}`[0..] | 原值 |
| `stop_conditions` | R，`{code,condition,return_to}`[1..]；code 唯一、condition 非空、return_to Actor；必须含原规格五项标准停止码 | 原值 |
| `next_owner`, `return_to` | R，Actor | 原值 |
| `decision_sources` | R，`{source,actor,decided_at}`[1..]；actor Actor、时间为 RFC3339 | 原值；客户端注明的决定者/时间不是平台认证或人类签署 |
| `change` | R，`{reason,source_refs}`，非空文本及 Source[1..] | 原值 + 只读 actor/recorded_at/from_version/to_version/changed_fields |
| `contract_id`,`issue_id`,`workspace_id`,`version`,`contract_required` | **禁止输入** | 平台生成/绑定；version 正整数，contract_required=true |
| `versions.issue_basis` | **禁止输入** | `{revision,title_digest,description_digest}`；发布 CAS 的实际任务内容指纹 |

`Action={id,operation,resources,acceptance_ids}`，operation 为 read/edit/test/deliver；resources 为 Resource[1..]，acceptance_ids 为已存在 acceptance ID 的非空无重复数组。`DeniedAction` 同结构，acceptance_ids 允许 `[]`（禁止本身不需要正向验收），其余不变；禁用项不因 acceptance 为空被忽略。

`versions={code_basis,document_basis,code_not_applicable_reason?}`：
- code_basis 为 `{repository,commit_sha,branch,dirty_snapshot}`[0..]；repository 是完整固定仓库 URL（同一合同内精确相等、无凭据/query/fragment），commit_sha 为 40 或 64 位小写完整 Git object ID；branch 为非空说明，不用于匹配；dirty_snapshot 必填，干净为 null，否则为 kind=artifact 的完整 Source，指向可恢复补丁/文件快照，不是裸摘要。
- code_basis.repository 必须唯一；重复在发布时 409 contract_conflict，路径为第二项的完整 Pointer（如 `/versions/code_basis/1/repository`），reason=duplicate_id。code_basis=[] 时必须有非空 code_not_applicable_reason；有代码项时禁止该字段。document_basis 为 `{source,applicability}`[1..]，至少包含主需求；source 为完整 Source，applicability 非空。issue_basis 的 revision 只作发布证据；开工只比内容指纹，不比聚合 revision。

`Resource={kind,locator,match}`：issue/comment 来源权限不因 Resource 扩大；Resource.kind 仅 repository/issue/document/artifact。issue/artifact 的 locator 为 UUID，document 为 §1.3 的仓库路径对象，repository 为 `{repository,path}`；match 仅 exact，repository 才可 directory。path 为仓库根相对 POSIX 路径（根目录只用 `.`），大小写敏感；拒绝空值、绝对路径、反斜线、`.`/`..` 路径段（单独根 `.` 除外）、重复 `/`、尾 `/`、NUL、glob/正则。不得 URL 解码后再作为不同路径解释；目录匹配按完整段前缀，不让 `src` 匹配 `src2`。不做通用本地工具防火墙承诺。

**成功：** 初建 201，变更 200；body 就是完整快照，不再包 envelope。GET I、GET I/versions/{v} 返回同一持久快照形状，历史内容不因当前版本变化被改写。PUT 成功的 change 中：actor 取平台认证；from_version 首次 0；to_version 为新版本；recorded_at 为服务端时间；changed_fields 为排序的已变更顶层业务字段 Pointer，嵌套变化归其顶层，排除 change 和所有服务端字段。首次列出所有初建业务字段。

- 发布时锁内同时检查 expected_version 和任务 revision；二者都冲突时优先 contract_version_conflict；无变化也必须先过 CAS。
- 比较全部可写业务字段（排除 change）经 JCS 后相等即 no-op：200 返回原快照，既不加版本/变更，也不替换 change.reason/source_refs。首次采用、当前指针、历史、变更、旧 queued 失效均按原规格原子提交。发布不改任务正文、assignee 或 Issue status。
- 阻断问题/假设可合法记录并发布，**阻止 handoff/start，不阻止结构完整的发布**。已失效、不可读或摘要不符的必需来源则不能发布；不能用空内容冒充有效材料。
- 版本列表 `GET I/versions?limit=20&cursor=...` 返回 `{items:[{version,change}],next_cursor:null|string}`，上限 100，版本倒序稳定游标，不携带历史全文。

### 1.3 Source 与 source_refs

**唯一形状：** `Source={kind,locator,version,digest,supplied_by}`，五项必填、非 null；supplied_by 为有效 Actor，表示材料提供者，不代替请求认证。所有 `source_refs`、`goal_sources`、`evidence_refs` 都是**完整 Source 对象数组**，不是字符串、input ID、JSON Pointer、URL 或新的 Source 注册表。

| kind | locator | version | digest 的准确字节 |
| --- | --- | --- | --- |
| issue | 任务 UUID | 当次读取的正整数 revision | `JCS({"title":原始标题,"description":原始描述或null})`；不含 revision/status/评论/附件/更新时间 |
| comment | 评论 UUID | 该评论正整数 revision，不是任务 revision | 该评论原始 content 的 UTF-8；不含作者/时间/换行补全 |
| document | `{repository,path}`，固定仓库与相对文件路径 | 完整 commit SHA | 对应 commit 中该普通文件的原始 blob 字节；不采用 checkout 换行转换；不解引用 symlink |
| artifact | 附件 UUID | 该附件 UUID（本批不可变内容身份；替换必须新 ID） | 下载文件原始字节，不含 HTTP/存储元数据；压缩包摘要针对包文件，不针对展开目录 |

**没有** CRLF→LF、Unicode NFC/NFKC、Markdown 渲染、trim、BOM 剥离、JSON 字符串转义或自动尾换行规范化（只有 issue 的对象用 JCS）。每个 Source 必须按现有权限定位具体资源、核对归属、读取并核验；不可读对象不返回其存在性/名字/内容。仓库/附件权限不因提供 digest 而跳过。平台不能读取的本地固定文件须先有可访问固定附件或受认证运行时观测，不凭 URL/客户端文本声称已验证。

解析规则：无效 kind/locator/version/digest 类型及形状 → 400 invalid_request；格式正确但资源不存在/不可读 → 422 input_unavailable（仅向已授权的任务调用者给安全字段路径，不泄露目标详情）；实际 digest/version 不符 → 409 source_changed。同一 kind+locator+version 在各处可重复，digest 必须相同；互相矛盾 → 409 contract_conflict，按 §4 稳定 Pointer 顺序列出不一致字段。无需先出现在 effective_inputs 才能引用，但主需求必须同时属于 effective_inputs/document_basis。未知 acceptance/action ID → 409 contract_conflict；越界 planned action ID 另按 §2。所有引用不得隐式跟随“最新”。

**issue 特例：** 首次引用 issue 或更换其内容指纹时，发布须核对 Source.version 为当前读到的 revision。后来发布若沿用此前已成功验证的同一 issue Source（同 locator/version/digest），即使普通评论/状态/附件使聚合 revision 增长，只要 title/description digest 未变就允许保留旧 Source.version，服务端不得强制刷新；expected_issue_revision 仍须是本任务当前值。调用者也可显式读取并更新所有同源引用到新 revision，此为来源证据变化，成功发布会生成新合同版本。发布后 handoff/claim/start 只比内容摘要，不修改持久 Source.version 或 input_digest。引用 comment 的 revision 则必须精确不变，即使编辑回原文也不能冒充原决定。`issue_basis.title_digest=H(JCS(title))`，description_digest=H(JCS(description|null))。这避免普通评论自动失效，同时保留显式来源更新的可追溯性。

可复现向量（字符串代码块中的 `\n`/`\r` 表示对应真实字节，不是反斜线字符）：

| 输入 | SHA256 |
| --- | --- |
| UTF-8 `abc`（hex `616263`） | `ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad` |
| UTF-8 `abc\n` | `edeaaff3f1774ad2888673770c6d64097e391bc362d7d6fb34982ddf0efd18cb` |
| UTF-8 `abc\r\n` | `552bab6864c7a7b69a502ed1854b9245c0e1a30f008aaa0b281da62585fdb025` |
| issue JCS `{"description":null,"title":"abc"}` | `8f32d28e013043aaa07ef670f216566e07c1643ff1a0cea8fcf644b66543ca59` |
| JCS `"abc"`（含 JSON 双引号） | `6cc43f858fbb763301637b5af970e2a46b46f461f27e5a0f41e009c59b827b25` |
| JCS `null` | `74234e98afe7498fb5daf1f36ac2d78acc339464f950703b8c019892f982b90b` |

这些是摘要算法向量，不代替合法任务标题/完整快照。MOSS 既有 `w54Digest(description)` 及 issue version=1 是候选 builder，需改为实际内容投影及读到的 revision；不是修改验收标准。

### 1.4 最小合法 PUT 示例

以下全部是合成 UUID，不映射平台真实人/任务。夹具预先创建同工作区任务 `100…001`（title=`abc`、description=null、revision=1）、有效成员 `200…001`；Source 中的 issue 正是该任务。真实测试须按实际 seed 的 revision 构造，不假定所有数据库从 1 开始。

```json
{
  "expected_version": 0,
  "expected_issue_revision": 1,
  "schema_version": 1,
  "goal": "读取已批准输入并交付版本证据",
  "goal_sources": [{"kind":"issue","locator":"10000000-0000-4000-8000-000000000001","version":1,"digest":"8f32d28e013043aaa07ef670f216566e07c1643ff1a0cea8fcf644b66543ca59","supplied_by":{"type":"member","id":"20000000-0000-4000-8000-000000000001"}}],
  "allowed_actions": [{"id":"read-input","operation":"read","resources":[{"kind":"issue","locator":"10000000-0000-4000-8000-000000000001","match":"exact"}],"acceptance_ids":["a1"]}],
  "prohibitions": {"text":"不修改产品、不触发下游运行","denied_actions":[]},
  "acceptance": [{"id":"a1","assertion":"证据列出来源固定版本和摘要","evidence_required":["test-log"],"source_refs":[{"kind":"issue","locator":"10000000-0000-4000-8000-000000000001","version":1,"digest":"8f32d28e013043aaa07ef670f216566e07c1643ff1a0cea8fcf644b66543ca59","supplied_by":{"type":"member","id":"20000000-0000-4000-8000-000000000001"}}]}],
  "effective_inputs": [{"id":"input1","source":{"kind":"issue","locator":"10000000-0000-4000-8000-000000000001","version":1,"digest":"8f32d28e013043aaa07ef670f216566e07c1643ff1a0cea8fcf644b66543ca59","supplied_by":{"type":"member","id":"20000000-0000-4000-8000-000000000001"}},"purpose":"主需求","required":true}],
  "confirmed_facts": [],
  "assumptions": [],
  "versions": {"code_basis":[],"code_not_applicable_reason":"仅阅读合成任务","document_basis":[{"source":{"kind":"issue","locator":"10000000-0000-4000-8000-000000000001","version":1,"digest":"8f32d28e013043aaa07ef670f216566e07c1643ff1a0cea8fcf644b66543ca59","supplied_by":{"type":"member","id":"20000000-0000-4000-8000-000000000001"}},"applicability":"主需求"}]},
  "deliverables": [{"id":"d1","description":"版本证据","acceptance_ids":["a1"],"evidence_refs":[]}],
  "open_questions": [],
  "stop_conditions": [
    {"code":"input_unavailable","condition":"缺有效输入即停止","return_to":{"type":"member","id":"20000000-0000-4000-8000-000000000001"}},
    {"code":"input_version_mismatch","condition":"版本不符即停止","return_to":{"type":"member","id":"20000000-0000-4000-8000-000000000001"}},
    {"code":"contract_out_of_scope","condition":"结构化动作越界即停止","return_to":{"type":"member","id":"20000000-0000-4000-8000-000000000001"}},
    {"code":"permission_denied","condition":"授权不足即停止","return_to":{"type":"member","id":"20000000-0000-4000-8000-000000000001"}},
    {"code":"blocking_question","condition":"存在阻断问题或假设即停止","return_to":{"type":"member","id":"20000000-0000-4000-8000-000000000001"}}
  ],
  "next_owner": {"type":"member","id":"20000000-0000-4000-8000-000000000001"},
  "return_to": {"type":"member","id":"20000000-0000-4000-8000-000000000001"},
  "decision_sources": [{"source":{"kind":"issue","locator":"10000000-0000-4000-8000-000000000001","version":1,"digest":"8f32d28e013043aaa07ef670f216566e07c1643ff1a0cea8fcf644b66543ca59","supplied_by":{"type":"member","id":"20000000-0000-4000-8000-000000000001"}},"actor":{"type":"member","id":"20000000-0000-4000-8000-000000000001"},"decided_at":"2026-10-29T00:00:00Z"}],
  "change": {"reason":"建立本阶段合同","source_refs":[{"kind":"issue","locator":"10000000-0000-4000-8000-000000000001","version":1,"digest":"8f32d28e013043aaa07ef670f216566e07c1643ff1a0cea8fcf644b66543ca59","supplied_by":{"type":"member","id":"20000000-0000-4000-8000-000000000001"}}]}
}
```

201 成功 body = 上述完整请求删去两个 expected 字段，保留其余业务字段，再加入以下顶层字段，并分别向原 `versions`、`change` 对象加入下列同名对象的成员（**递归合并的响应字段片段，不是独立完整响应，不替换原 versions/change**）。没有辅助伪键；GET 返回同一持久结果。

```json
{
  "contract_id":"30000000-0000-4000-8000-000000000001",
  "issue_id":"10000000-0000-4000-8000-000000000001",
  "workspace_id":"40000000-0000-4000-8000-000000000001",
  "version":1,
  "contract_required":true,
  "versions":{"issue_basis":{"revision":1,"title_digest":"6cc43f858fbb763301637b5af970e2a46b46f461f27e5a0f41e009c59b827b25","description_digest":"74234e98afe7498fb5daf1f36ac2d78acc339464f950703b8c019892f982b90b"}},
  "change":{"actor":{"type":"member","id":"20000000-0000-4000-8000-000000000001"},"recorded_at":"2026-10-29T01:00:00Z","from_version":0,"to_version":1,"changed_fields":["/acceptance","/allowed_actions","/assumptions","/confirmed_facts","/decision_sources","/deliverables","/effective_inputs","/goal","/goal_sources","/next_owner","/open_questions","/prohibitions","/return_to","/schema_version","/stop_conditions","/versions"]}
}
```

非法完整请求的最小变体：在上例仅替换 acceptance[0].source_refs 为以下 JSON（字符串引用不再猜测为 effective_input）：

```json
["input1"]
```

结果 400 invalid_request，field_errors[0].path=`/acceptance/0/source_refs/0`，reason=`invalid_type`；合同、采用标记、历史、队列均不变，N 成立。其他确定性非法变体见 §4。

## 2. handoff、accept 与持久事实（WS-02）

### 2.1 显式交接

`POST I/handoffs` 请求五项：current_version 正整数；planned_action_ids 非空无重复数组；resources 非空 Resource 数组；target Actor；idempotency_key 非空字符串（不含凭据，作用域沿原规格）。不允许自报 handoff_id、task_id、input_digest、accepted_at、state 或 receipt。

资源在请求中是该计划统一使用的资源集合；服务端把每个 action_id 解析成合同中的 operation，并对每个 action 与每个请求资源做权限/允许/禁止判断，生成 `planned_actions=[{action_id,operation,resources}]`。若各动作资源不同，拆成不同交接而非猜隐式映射；本补充不引入另一个输入格式。未知 action → contract_out_of_scope；目标不是 next_owner → receiver_mismatch；已知动作但超资源范围或命中禁止 → contract_out_of_scope。非法路径 400，不自动纠正后执行。

最小合法请求（采用 §1.4 v1、成员目标）：

```json
{"current_version":1,"planned_action_ids":["read-input"],"resources":[{"kind":"issue","locator":"10000000-0000-4000-8000-000000000001","match":"exact"}],"target":{"type":"member","id":"20000000-0000-4000-8000-000000000001"},"idempotency_key":"member-v1-first"}
```

201 body **直接返回 handoff 对象**。同键同有效载荷且原版本仍当前有效时重放 200，同 ID、不再派工；原版本已替换则 409 contract_stale，返回原 handoff_id/task_id 与 current_version，不把旧 accepted 当当前成功；同键异载荷 409 idempotency_conflict。三者均先过当前访问权限门；载荷摘要为去掉 idempotency_key 后完整请求的 H(JCS)，数组顺序保留。已记录的同键拒绝返回原拒绝 ID/结果，不重复写入或派工；若输入/版本已改变，仍须显式新键重新交接，不能把失败记录变为旧请求的成功。

```json
{
  "handoff_id":"50000000-0000-4000-8000-000000000001",
  "issue_id":"10000000-0000-4000-8000-000000000001",
  "version":1,
  "state":"pending",
  "target":{"type":"member","id":"20000000-0000-4000-8000-000000000001"},
  "return_to":{"type":"member","id":"20000000-0000-4000-8000-000000000001"},
  "planned_actions":[{"action_id":"read-input","operation":"read","resources":[{"kind":"issue","locator":"10000000-0000-4000-8000-000000000001","match":"exact"}]}],
  "created_at":"2026-10-29T01:01:00Z",
  "created_by":{"type":"member","id":"20000000-0000-4000-8000-000000000001"},
  "task_id":null,
  "dispatch":null,
  "receipt":null,
  "latest_admission_id":null
}
```

智能体目标使用相同形状；成功入队的 dispatch 示例为 `{"status":"queued","reason_code":"queued","target":{"type":"agent","id":"20000000-0000-4000-8000-000000000002"},"task_id":"60000000-0000-4000-8000-000000000001","run_id":"60000000-0000-4000-8000-000000000001"}`，handoff.task_id 同值，但仍 pending、receipt=null。沿用既有 queued/coalesced/deferred/blocked 等 dispatch 字段语义，不新增“accepted”派工状态、不以 queued 冒充开工。deferred 时可无运行；member 不产生 dispatch 或运行。

合同同步校验拒绝返回 §4 非 2xx，不创建新运行；已授权且格式可解析的交接业务拒绝保存 admission，即使无 handoff_id 也能读取。若创建 pending 后既有派工被拒，返回非 2xx、同一 handoff 的拒绝状态及 dispatch.status=blocked，记录 admission；不暗藏失败在 201。入队/去重结果必须持久且同键可恢复，不能存在成功响应但无可恢复交接、孤立未绑定运行或重放二次派工。受保护合同不允许用 steered 将新确认塞入正在运行的回合。

最小非法请求（只改已知合法请求的 planned_action_ids）：

```json
{"current_version":1,"planned_action_ids":["unapproved-edit"],"resources":[{"kind":"issue","locator":"10000000-0000-4000-8000-000000000001","match":"exact"}],"target":{"type":"member","id":"20000000-0000-4000-8000-000000000001"},"idempotency_key":"member-v1-bad-action"}
```

409 contract_out_of_scope、out_of_scope，路径 `/planned_action_ids/0`；授权调用者可读持久 admission；同步无运行增量、无 assignee/status/通知变化，N。

### 2.2 成员 accept 与回执

`POST I/handoffs/{handoff_id}/accept` 只接受 `{"current_version":1}`。必须是 target 本人认证；target/next_owner 仍一致、来源/内容/权限/计划仍有效。无 task_id/generation/runtime 观测要求；代码活动仍须合同中可读的固定代码材料，成员回执不声称平台读取了其本地电脑。

200 body **直接返回 receipt 对象**，与 GET handoff 的 receipt 完全相同；下例 input_digest 已按 §1.4 合成快照及 §3.3 算出。真实 seed 改变 contract_id/来源后须重新计算，不能照抄常量：

```json
{
  "handoff_id":"50000000-0000-4000-8000-000000000001",
  "version":1,
  "target":{"type":"member","id":"20000000-0000-4000-8000-000000000001"},
  "task_id":null,
  "claim_generation":null,
  "runtime_id":null,
  "input_digest":"eefb7d3a7502508494dadeb5c2940d515bd0baededed7468e179170122137094",
  "planned_actions":[{"action_id":"read-input","operation":"read","resources":[{"kind":"issue","locator":"10000000-0000-4000-8000-000000000001","match":"exact"}]}],
  "accepted_by":{"type":"member","id":"20000000-0000-4000-8000-000000000001"},
  "accepted_at":"2026-10-29T01:02:00Z"
}
```

accepted_by/accepted_at 平台生成；重复本人 accept 在仍有效的同版本返回同回执，不追加接手事实。新版本/来源改变后重放返回 stale；原回执仍作为历史事实保留。非本人 403 permission_denied；智能体目标的这个端点对任何 Web/成员请求均 403 agent_accept_via_runtime_only。只要 target 是 agent，就不能通过请求体改成 member 代签。

最小非法 accept（试图伪造运行绑定）：

```json
{"current_version":1,"task_id":"60000000-0000-4000-8000-000000000001"}
```

400 invalid_request，路径 `/task_id`、reason=read_only_field；不产生回执或运行，N。对 agent handoff 的合法形状请求仍先受上述 runtime-only 权限限制，不能借错误体探测私有 agent。

### 2.3 pending / accepted / 拒绝读取入口

补齐原规格“拒绝刷新后可查”的最小 App API 读取投影，不是新增看板/审计平台：

| 入口 | 200 body | 说明 |
| --- | --- | --- |
| `GET I/handoffs?limit=20&cursor=...` | `{items:[handoff],next_cursor:null|string}` | 当前与历史，created_at+ID 倒序，max100 |
| `GET I/handoffs/{id}` | handoff 对象 | state、原绑定 version、receipt（nullable）、latest_admission_id（nullable）；历史不当当前成功 |
| `GET I/admissions?limit=20&cursor=...&handoff_id=...&task_id=...` | `{items:[admission],next_cursor:null|string}` | 两个过滤器可省略；包括创建交接前的业务拒绝；checked_at+ID 倒序，max100 |
| `GET I/admissions/{id}` | admission 对象 | 错误响应中的 admission_id 可直接定位 |

admission 必含：admission_id、issue_id、request_id、operation（handoff/member_accept/claim/start）、outcome（accepted/rejected）、code（成功为 null）、handoff_state、field_errors（成功为空数组）、expected_version（nullable）、checked_version（nullable）、current_version（nullable）、checked_at、return_to（缺有效合同时 nullable）、handoff_id/task_id/claim_generation/runtime_id（均按是否已存在 nullable）、input_digest（尚未算出时 null）、planned_actions。成功 handoff/claim 的事实由 handoff/运行持久记录表达，不额外创建 outcome=accepted 的 admission；只有实际 member_accept/start 成功才保存 accepted admission。各阶段业务拒绝均保存 rejected admission。restart/rerun/retry/wakeup/resume 的准入事实按实际裁决入口记 claim 或 start，不设 recovery 枚举。字段集不因进程重启丢失；不在错误/日志中持久凭据或原始私有输入。start admission 另含 code_observations（无代码时 []），并只读记录认证 runtime_id 与 claim_generation；这是认证通道收到的元数据观测，不是用户自报观测者。其他 operation 的 code_observations 为 null。

只有已授权到该任务及相应记录资源的调用者能看这些投影；读接口不得重新校验后改 state、创建拒绝、派工或通知。未授权请求不写跨租户合同事实，只可进入既有受限安全日志。权限撤回后的读取遵守现有非枚举规则，不能凭 admission_id 取回旧私有内容。无权查看记录所引用资源时，按既有非枚举规则拒绝该记录读取，不返回含私有字段的半份成功响应；不能任意把本补充非 nullable 字段置 null 当作合法结果。

handoff 的 state 是最新业务结论投影，accepted receipt 是不可变历史证据：发布 v2 使 v1 pending 变 stale；旧 accepted 不改写成 v2 已接手。若后续旧确认重放被拒，只增加 rejection admission，不销毁原 receipt。不以状态投影重建第二套 Run 生命周期。

## 3. 运行时能力、认证观测与不可伪造绑定（WS-03）

### 3.1 能力与认证来源

新增能力字符串固定为 `task-contract-v1`，**只表达能完整执行本补充的受保护任务握手**。复用：运行时注册请求已有 capabilities 数组、持久 metadata.capabilities；HTTP `X-Client-Capabilities` 及 WS 建连捕获的 capability；start 现有 capabilities 数组。受保护任务 claim 须该 runtime 的持久声明及当前连接声明均含该值；start 还须本次能力声明包含。仅版本号、prompt 文本、客户端 UI 标记或注册中自报新 runtime_id 都不能证明能力。缺/未知能力 409 contract_protocol_unsupported，不降级 legacy start；普通未采用任务保持原路径。

身份来自既有平台认证：mdt_ 的 daemon/owner/workspace → runtime 归属 → task.runtime/agent/workspace；现有 Access 在 handoff、claim、start 各边界重新校验。mat_ 仅证明所绑定 task/agent/workspace，不自动成为 runtime 观测权限；普通智能体业务请求或成员凭据不得代交运行时观测。即使相同成员可管理 runtime，也必须走现有 daemon 认证通道并证明该 runtime/claim 的归属，不接受浏览器 POST 自称 runtime。

不增加角色或授权；不把合同行文、supplied_by、decision_sources.actor 或 HTTP 请求体里的 target 当成身份认证。请求自造 `X-Agent-ID`、`X-Task-ID`、workspace 或 receipt 均不能覆盖认证上下文。

### 3.2 claim/start 的 wire 接入

不新建 claim/start 公共命令或通道：
- claim **请求**保持现有形状（单 runtime `{}`；批量 `{daemon_id,runtime_ids,max_tasks}`），不得输入 contract_admission；声明能力在既有 header/连接中传递。claim **响应**单项仍 `{task:...}`，批量仍 `{tasks:[...],...既有poll字段}`，HTTP/WS 相同；各受保护 task 响应只增 `contract_admission`，未采用任务可省略。该对象精确字段为 `{status:"pending",contract:完整快照,handoff_id,version,input_digest,planned_actions,code_basis,runtime_id,task_id,claim_generation,receipt:null}`；code_basis 等于 contract.versions.code_basis，另外三项运行绑定须等于外层 task，contract 无 data/snapshot 包裹层。所有这些字段均服务端生成，不能从 claim 请求覆盖。
- `claim_generation` 是现有 `dispatched_at` 的原样 RFC3339Nano 字符串（数据库微秒精度），不是新计数器；服务端精确比较 task_id/runtime_id/dispatched_at。不能根据当前时间自行构造。无运行的 member 必为 null。
- claim 成功只令已有运行 dispatched，contract_admission.status=`pending`，**没有 accepted 回执或可执行许可**。携带当前完整合同（不截断）、handoff_id、version、input_digest、planned_actions、code_basis、runtime_id、task_id、claim_generation；input_digest 为服务端算出的预期输入承诺，不是 bearer token。claim 失败的队列项按原规格终结 cancelled，不生成 retry；不得通过空 tasks 使拒绝事实失联，须可从该 task 的 admission 查询/现有 reason_code 定位。

下面为 start **完整最小请求**（沿 §1.4 的相同固定输入，另建已授权的 agent 目标/运行/handoff；目标本身不进入 input_digest 投影；无代码活动时 code_observations 必须 `[]`，不是省略）：

```json
{
  "runtime_id":"70000000-0000-4000-8000-000000000001",
  "dispatched_at":"2026-10-29T01:03:00.123456Z",
  "capabilities":["task-contract-v1"],
  "contract_admission":{
    "handoff_id":"50000000-0000-4000-8000-000000000002",
    "version":1,
    "input_digest":"eefb7d3a7502508494dadeb5c2940d515bd0baededed7468e179170122137094",
    "planned_actions":[{"action_id":"read-input","operation":"read","resources":[{"kind":"issue","locator":"10000000-0000-4000-8000-000000000001","match":"exact"}]}],
    "code_observations":[]
  }
}
```

路径 `/api/daemon/tasks/{task_id}/start` 已绑定 task，嵌套不接收 task_id、target、workspace_id、accepted_at 或 claim_generation 副本（generation 只取外层 dispatched_at）。`version/input_digest/planned_actions` 必须逐项等于平台当前持久 handoff/claim，不接受客户端替换后重算；input_digest 错不等于身份问题，返回 409 input_version_mismatch。

确定性绑定错误：handoff_id 不等于该 task 的持久绑定 → 409 contract_stale，`/contract_admission/handoff_id`；合法格式但过期 generation → 409 contract_stale，`/dispatched_at`；planned_actions 任一值或集合不同 → 409 contract_out_of_scope，`/contract_admission/planned_actions`。code_observations 缺少必需仓库项、增加未声明项或重复仓库 → 409 input_version_mismatch，分别定位数组根、额外项/重复项的 `/repository`（如 `/contract_admission/code_observations/1/repository`）；缺字段/null类型非法仍 400。缺失/不可读的 dirty 附件 → 422 input_unavailable，`/contract_admission/code_observations/0/dirty_snapshot`；可读但摘要/干净状态不同 → 409 input_version_mismatch，同一 Pointer。以上先过 runtime 归属鉴权，不泄露别的 handoff/claim 是否存在。

有代码基线时每一 repository 恰好一个观测：

```json
{"repository":"https://example.invalid/synthetic.git","commit_sha":"a34817f8faa8b0262a538552c02f12540ce6680c","dirty_snapshot":null}
```

这是 code_observations 中的合法**单项形状**，其 SHA 必须来自对应测试仓库实际读取。dirty 时 dirty_snapshot 是 §1.3 artifact Source；附件含可恢复快照且 digest 与合同一致。观测集不许缺失、重复、增加未声明仓库。分支名不用作 SHA；干净/dirty 不可互换。不能拿守护进程自动生成的工作树基线提交冒充获批用户基线；若环境有合成基线，应能回溯已读取的批准 commit 与 dirty 快照，只比较批准的输入。

观测只在获认证的目标 runtime 内、该 task/generation 的元数据准备阶段读取，先于模型/业务命令；只读 Git 元数据和文件摘要、不调用项目 hooks/build/test、不自动 fetch/checkout/commit/上传源码。已有 fixed artifact 读取走既有权限；缺少可核验 dirty 附件时拒绝并接回，不在准入前悄悄上传真实数据。平台能验证认证来源、绑定与内容一致性，**不能证明被攻陷 runtime 没说谎，也不能远程读取任意本地目录**；不引入 TPM、签名代理或通用 runtime 改造。

start 200 沿用现有任务响应顶层，仅增 `contract_admission:{status:"accepted",receipt:Receipt}`。agent Receipt 与 §2 相同，target/accepted_by 为认证 agent，task_id/runtime_id 非 null，claim_generation 非 null且等于 dispatched_at。在同一裁决中持久 receipt/admission，并让现有运行可启动；没有此确认，本地不得执行。legacy 空 200、缺 receipt、未知状态、绑定不符即不可启动，不能据 HTTP 200 一项放行。

最小非法 start 变体（其余沿合法请求，只把 `/contract_admission/version` 改为 2；以下为**变体字段片段，不替换整个对象**）：

```json
{"version":2}
```

若真实 handoff 当前为 v1，替换 version=2 后返回 409 contract_stale，路径 `/contract_admission/version`；原 queued/dispatched 项按原规格 cancelled，从未启动、无新运行/自动 retry，N。缺 code_observations → 400 invalid_request；代码项 SHA 与批准值不符 → 409 input_version_mismatch，路径 `/contract_admission/code_observations/0/commit_sha`。非归属 runtime 即使值全猜中仍先 403/非枚举404，无合同详情泄露。

同一 task/generation 的完全相同 start 重放在仍有效且既有启动结果允许重放时返回原 receipt，不再次授予本地进程启动；守护进程也按 task+generation 单次执行。旧 generation 不得取消/改写后来 claim 的当前运行。重启、rerun/retry/wakeup/resume 必须由当前入口重新准入，不能复制旧回执作为新执行的许可。

### 3.3 input_digest 的确定投影

服务端在 handoff/claim/start 都从实际已授权读取和持久记录重建，不信任客户端传入 digest 证明输入有效：

`input_digest = H(JCS({schema_version:1,contract_id,version,issue_basis:{title_digest,description_digest},sources,code_basis}))`。

其中：
- sources 为快照所有完整 Source 出现位置（含 change/decision/evidence/document_basis/dirty_snapshot）按 kind+locator+version 去重后的 `{kind,locator,version,digest}` 数组；不含 supplied_by、显示名、时间和查询顺序；按每项 JCS 字节升序排列。issue Source 的 version 固定采用发布快照中的值，后续只校验内容摘要，不能因普通评论改写此投影。
- code_basis 为 `{repository,commit_sha,dirty_snapshot}` 数组，按 repository 的 UTF-8 升序；dirty_snapshot 为 null 或 `{kind,locator,version,digest}`，去掉 supplied_by。branch 不是输入身份。
- issue_basis 不含聚合 revision；不含 checked_at、observed_at、request_id、运行日志、历史全文。合同 version 绑定不可变快照，planned_actions 则**单独绑定**回执和 claim，不能以 digest 相同为由替换计划。
- 观测与固定 Source 全部匹配后才接受该承诺；任何访问/校验暂不可用失败关闭。digest 不含可识别秘密，不作为跨工作区缓存/查找授权键。

§1.4 示例按该投影所得 input_digest 为 `eefb7d3a7502508494dadeb5c2940d515bd0baededed7468e179170122137094`。MOSS 可以独立计算该投影作为期望值，但不得将它写成测试自有的“产品准入服务”。§1.3 算法向量可先验证规范化与编码；正式回执值须用实际 seeded 快照计算，不使用示例常量。

## 4. 稳定错误、持久性与权限（WS-04）

沿用 App API 顶层英文 `error`，增加 code 等；不采用 Public API envelope。授权业务拒绝最小完整示例（当前合法任务 v2，旧页面按 v1 交接）：

```json
{
  "error":"the contract version is stale",
  "code":"contract_stale",
  "handoff_state":"stale",
  "field_errors":[{"path":"/current_version","code":"contract_stale","reason":"version_mismatch"}],
  "expected_version":1,
  "checked_version":1,
  "current_version":2,
  "return_to":{"type":"member","id":"20000000-0000-4000-8000-000000000001"},
  "request_id":"80000000-0000-4000-8000-000000000001",
  "admission_id":"90000000-0000-4000-8000-000000000001",
  "handoff_id":null,
  "task_id":null,
  "checked_at":"2026-10-29T01:04:00Z"
}
```

field_errors[] 的 code 为稳定顶层业务码，reason 为稳定局部原因（required/invalid_type/invalid_format/read_only_field/unknown_field/duplicate_id/unresolved_reference/version_mismatch/digest_mismatch/path_escape/permission_denied/service_unavailable）。无字段原因用根 Pointer。重复项/矛盾来源按所有候选完整 Pointer 的 UTF-8 字节升序检测，第一项为基准，返回所有不一致项的完整 Pointer，不能仅写 `/id` 或 `/digest`。请求格式错误/发布错误可以没有 admission，此时 admission_id=null；发布结果本身由当前/历史查询核实，不强制将所有非法 JSON 持久成准入历史。

| HTTP / code | handoff_state | 典型 Pointer / 确定性最小非法变体 |
| --- | --- | --- |
| 400 invalid_request | needs_clarification | 在完整 PUT 增 `"version":1` → `/version` read_only_field；`expected_version:null` → `/expected_version` invalid_type；未知字段同码 |
| 422 contract_incomplete | needs_clarification | 完整 PUT 删 goal → `/goal` required；acceptance=[] → `/acceptance` required；重复局部 ID → 如 `/allowed_actions/1/id` duplicate_id（按下述稳定 Pointer 顺序） |
| 409 contract_version_conflict / revision_conflict | conflict | `/expected_version` / `/expected_issue_revision`；不切当前、不追加历史 |
| 409 contract_run_active | conflict | 根 Pointer；现有 dispatched/running/waiting_local_directory 或仅请求取消仍活跃；不发布 |
| 409 contract_stale | stale | `/current_version`、`/contract_admission/version`；旧幂等请求亦不能变当前成功 |
| 409 source_changed | stale | `/effective_inputs/0/source/digest` 等具体 Source；原 issue title/description 变化也定位 `/versions/issue_basis` |
| 409 input_version_mismatch | stale | `/contract_admission/input_digest` 或具体 code_observations 字段 |
| 409 contract_conflict | conflict | `/allowed_actions/0/acceptance_ids/0`=`"missing-a"`；同一来源身份的 digest 矛盾返回完整 Pointer，如 `/confirmed_facts/0/source_refs/1/digest` |
| 409 receiver_mismatch | conflict | `/target/id`；有权限可见但不是 next_owner 的目标 |
| 409 contract_out_of_scope | out_of_scope | `/planned_action_ids/0`、`/resources/0`；合法格式但未获批动作/资源、禁止优先 |
| 400 invalid_request | needs_clarification | `/resources/0/locator/path`=`"src/../private"` → path_escape；拒绝而非规范化后执行 |
| 422 input_unavailable | needs_clarification | 实际 Source Pointer；不存在/失效/无权材料统一安全信息，不返回材料名字或实际内容 |
| 422 blocking_question | needs_clarification | `/open_questions/0` 或 `/assumptions/0`；blocks_start=true；可发布、不能接手 |
| 409 contract_protocol_unsupported | unavailable | `/capabilities`；缺 task-contract-v1 或未支持 schema；失败关闭 |
| 409 idempotency_conflict | conflict | `/idempotency_key`；同键异有效载荷，无额外派工 |
| 503 validation_unavailable | unavailable | 具体可见字段或根；存储/校验暂不可用，不假 accepted |
| 404 contract_missing | needs_clarification | 根；仅向有权访问任务者表明缺合同；采用标记不回退 |
| 401 unauthorized / 403 permission_denied / 404 not_found | 不暴露 | 沿既有非枚举门；无 return_to/version/私有目标/admission 内容 |
| 403 agent_accept_via_runtime_only | 不暴露其他私有字段 | 已授权可见的 agent handoff 被 Web 代签，不能回退成员签名 |

未经授权的错误最小 body 为 `{"error":"access denied","code":"permission_denied","request_id":"80000000-0000-4000-8000-000000000002"}`，不返回合同辅助字段。每次先认证、工作区/任务/目标权限，再查幂等记录或读取引用正文。不得靠是否命中键探测旧私有合同。多个故障时只要求稳定按此阶段优先级：认证/非枚举 → JSON形状 → CAS/当前版本 → 结构引用 → 来源/阻断 → 权限再检查/动作范围 → 持久提交；测试隔离单一错误，不依赖多个独立字段错误的遍历顺序。

**持久拒绝与存储故障：** 对已授权、形状有效且能确定所属任务的 handoff/accept/claim/start 业务拒绝，须先持久 admission，才能返回带 admission_id 的拒绝事实。持久失败只能 503，admission_id=null，不能返回虚构可查 ID。已有队列准入拒绝要同事务记录拒绝并 cancelled；事务失败原运行保持原非执行状态、不得本地启动、不自动建 retry，允许同一 claim 的准入请求在存储恢复后确认结果（不是新业务运行）。旧 generation 的攻击请求不得终结较新的 generation。

**副作用统一：** 发布/读/拒绝不改 Issue assignee/status，不发评论/主动通知、不自动派下游。同步拒绝运行增量 0；已有排队后拒绝只保留并终结原记录，无自动 retry；无模型/业务进程、源码写入、越权正文读取。网络丢响应是未知，不写成零；同键/同 generation 确认真实提交事实。所有状态均是附属事实，不新增 Issue/Run status。

## 5. 并发与故障测试接点需求（WS-05）

这里固定**测试能观察和控制的语义边界**，不指定 service 函数名、表结构、锁实现或 hook 代码。后续 JARVIS 接到产品内部路径，MOSS 通过既有 testutil/TestApiClient、假运行时与 spy 测试真实产品；不得由 MOSS 自建合同状态机、mock 掉待证明的事务、直接调用测试专有“假产品服务”。测试接点不得注册公共 HTTP/CLI 入口、绕过认证或在生产配置启用。

### 5.1 受控屏障与可观察点

| 语义接点名（测试标识） | 到达位置 / 允许的测试控制 | 产品事实断言 |
| --- | --- | --- |
| `publish.before_arbitration` | 解析/认证已完成，争用本任务发布/claim互斥裁决之前；暂停/释放 | 两写者可同时 ready，不靠 sleep；真正锁仍由产品取得 |
| `claim.before_arbitration` / `start.before_arbitration` | 已找到目标候选并认证，进入同一任务裁决之前；暂停/释放 | 不是收到 HTTP 就算 claim 成功 |
| `publish.before_commit` | 已在裁决内，所有版本/来源及活跃运行检查完成，原子写集提交前；暂停/故障 | 外部读只见旧快照；无部分采用/版本/队列变更 |
| `claim.before_commit` | claim 已在裁决内准备 queued→dispatched 和新 dispatched_at，交付前；暂停/故障 | 发布不能在该活跃事实提交后成功绕过 |
| `start.before_commit` | 验证 runtime/generation/合同/来源/代码观测完成，receipt+running 可执行事实提交前；暂停/故障 | 本地启动计数仍 0；无仅 receipt 或仅 running 的半提交 |
| `operation.after_commit_before_response` | 数据库提交已确定，HTTP body 尚未发送；可丢该响应，不能回滚提交 | 同键/同generation 返回原事实，不新派工/不重复启动 |

每个接点至少可携带 operation/request_id/issue_id、已有 handoff/task ID、version、generation、事务“未提交/已提交”标记；不暴露 token/原始私有材料。测试用 channel/latch 的 ready/release，清理中保证解除阻塞；超时仅作失败保护，不用延时推断锁顺序。只在已明确授权且使用合成资源时提供资源读取 spy。

**最小合法屏障配置形状**（仅测试 harness 描述，不是产品 wire）：

```json
{"operation":"publish","point":"before_arbitration","issue_id":"10000000-0000-4000-8000-000000000001","action":"pause"}
```

非法配置（不存在的接点）不得被静默忽略，测试初始化必须报错、不执行业务：

```json
{"operation":"publish","point":"after_partial_commit","issue_id":"10000000-0000-4000-8000-000000000001","action":"pause"}
```

这是测试配置错误，不映射成用户可调用的产品 API；产品请求格式合法时的并发/故障错误仍必须按 §4 稳定码断言，不能用 harness 自造码冒充。

### 5.2 TC-12 / TC-14 必须裁决出的结果

1. **发布先赢：** v1 handoff 对应运行 queued；暂停其 claim 在 before_arbitration，发布 v2 完成，再释放 claim。一次提交后：当前 v2、历史 v1完整、唯一 v2 change；所有同任务旧版 queued 原 ID cancelled/reason=contract_stale；对应 pending stale；不得新增 retry。claim 无可执行 v1，start 亦不能用旧绑定成功，N。
2. **claim先赢：** 阻住发布 before_arbitration，让 claim 真正提交 dispatched，再释放发布。发布 409 contract_run_active，当前仍 v1，无 v2/history/change；claim 仍非 accepted，只有随后合法 start 才有 receipt。本地进程仍不因 claim 自动执行。
3. **start先赢：** 运行已经 dispatched 或 waiting_local_directory 时，发布本来就必须 409（不能错误预期“发布仍可赢到 start 前”）。通过合法 start 的原子 accepted/running 后也一样。仅发取消请求仍拒；确认现有运行终态后才能发布 v2。
4. **两写者：** 同 expected_version/revision，经 ready 屏障一同释放，一成功一 409；版本只增 1、change 只增 1；锁内重新读前提，不能只在事务外预检查。失败草稿由 Web 保留，不在数据库补写失败稿。
5. **旧 generation：** 重派后 T 的 generation G2，重放 G1 start，只拒 G1；不能 cancelled G2、覆盖 G2 receipt 或启动第二进程。授权撤回、Source 编辑也应在 commit 前重验，不缓存 claim 的布尔通过。

### 5.3 故障注入点与原子边界

每次仅注入一次确定错误；失败命中必须有计数/错误证据，未命中不得当已验证。

| 点 | 注入与预期产品事实 |
| --- | --- |
| `source_validation.read` | 对已授权来源存储读失败：503 validation_unavailable，不把故障当空材料；私有未授权正文读取 spy=0；N |
| `publish.persist_snapshot` / `publish.persist_change` / `publish.persist_pointer_adoption` / `publish.invalidate_queued` / `publish.commit` | 分别模拟写或提交前确定失败：整个发布回滚，旧快照/指针/历史/change/采用标记/queued/handoff 与前值完全一致，无 v2 半提交；首次失败仍 contract_required=false |
| `handoff.persist_binding` / `handoff.persist_dedup_dispatch` / `handoff.commit` | 确定失败不能留下无绑定新运行或二次派工；503，同键重试恢复。若系统分阶段已有持久 pending，则可读 pending，但不得伪报 dispatch 成功；后续同键只能续同一个交接，不能另建一份 |
| `admission.persist_rejection` / `admission.cancel_queued` / `admission.commit` | 故障时不能“一半拒绝一半可执行”；503 且不返回虚构 admission_id；原 queued/dispatched 不执行、不新建 retry；恢复后用原绑定确认 |
| `accept.persist_receipt` / `accept.make_executable` / `accept.commit` | 确定失败没有 accepted receipt 或运行可执行事实；运行仍非执行，不调用业务进程；成功则两者同时可观察 |
| `operation.after_commit_before_response` | 仅丢成功响应：结果未知，重读及同键/同generation 复验；已提交的快照/交接/回执/运行各仅一份；不做“零副作用”误断言 |
| `commit.outcome_unknown` | 模拟连接在提交确认阶段中断：503/传输未知，不能断言回滚；通过原 expected 版本+读取（发布）、原幂等键（交接）、原task/generation（start）确认实际持久结果，不换键或造新运行 |

存储完全不可用时不能同时要求“拒绝已持久可查”；必须明确 503/未知且无执行，恢复后再确定事实。此处不是降低原拒绝可恢复要求，而是区分**已记录拒绝、未能记录故障、提交结果未知**，避免虚假证据。

测试最少从三处交叉证明：①真实 App API GET 当前/历史/handoff/admission 与重启重读；②隔离数据库只读计数/状态（原运行ID、cancelled 原因、无 retry/跨任务下游）；③真实运行路径的假进程 marker、合成源码前后 SHA256、认证资源正文读取 spy。记录所有相关 request/handoff/task/version/generation，不仅断言错误文案。spy 要有合法阳性对照，证明不是从未接线；不调用真实智能体或付费模型。

## 6. 追溯与交接边界

| 补充项 | 保持的 FR/A → FD → SP | MOSS 据此补的 TC 子项（不表示已通过） |
| --- | --- | --- |
| WS-01 完整PUT/Source/解析 | FR/A-01、02、04 → FD-01/02/04 → SP-01/02/04 | TC-01/02/03 嵌套/只读字段、TC-04 材料、TC-06 CAS、TC-07 聚合revision、TC-08 固定摘要/dirty |
| WS-02 handoff/receipt/持久查询 | FR/A-03、04、06 → FD-03/04/06 → SP-03/04/06 | TC-01/05/09/10/11、TC-15 丢响应、TC-16 刷新拒绝、TC-18 重启 |
| WS-03 runtime绑定 | FR/A-03、05 → FD-03/05 → SP-03/05 | TC-08/10/12/13/17，所有已有入口，不仅单start |
| WS-04 错误/权限/N | FR/A-03、04、07 → FD-03/04/07 → SP-03/04/07 | TC-02/04/05/08/09/10/11/15，逐项N与非枚举 |
| WS-05 并发/故障接点 | FR/A-02、03、07 → FD-02/03/07 → SP-02/03/07 | TC-06/12/14/15/18，确定竞态顺序、原子失败、结果未知 |

**方案与影响：** 仅补原 App API 和受保护任务握手所需语义，新增读投影落实原“持久可查”标准；复用能力/认证/dispatch/claim/start。没有新角色/权限、工具级防火墙、通用 runtime 改造、合同热换或第二批功能。

**验收：** 原 A/TC/N 不变；本附件先由 Asurada 核验再交 MOSS。一般字段/envelope/摘要选择在已有目标内收敛，无新增人工产品决策；若有人要求扩大授权、任意工具拦截、主动通知或通用能力改造，才单列变更，不能据本附件默认获准。

**测试：** MOSS 沿固定成果适配 builder、保留真实首败并补逐 TC；产品缺入口继续记真实红灯/局部待接入，不把本规范当产品实现。本次只做输入指纹、JSON示例/摘要向量及文档一致性核对；未跑产品/正式回归、浏览器、性能、真实智能体或部署。

**仍需决策项：无新增产品/权限/验收决策。** 仍需 Asurada 核验本需求补充、后续 JARVIS 给唯一契约及 hook 的实际符号映射、MOSS 正式测试接入；这些是后续责任交接，不是要求人类重新批准普通格式选择，也不构成整项停工条件。缺失 InFrame 规范不从别处补入；沿父任务当前批准的 Multica 基线执行。

只读证据定位（固定 base）：`server/internal/handler/admission.go:28-94`（已有dispatch）；`server/internal/handler/daemon.go:1949-1989`（能力）、`:4598-4637`（start认证/generation）；`server/pkg/db/queries/agent.sql:982-988`（现有start fencing）；`server/internal/daemon/client.go:484-527`（现有start envelope）。新增合同字段/能力/来源观测在该 base 尚不存在，不可写成已实现。

下一责任人：Asurada 核验附件并交回 MOSS。WUSH-59 交付后 in_review；不启动 JARVIS、WUSH-56、部署或真实智能体，不改其他任务状态，不产生仓库提交或 PR。
