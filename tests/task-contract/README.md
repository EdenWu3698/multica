# WUSH-54 正式测试首轮证据（部分交付）

本目录由 MOSS 维护。依据 W54-SPEC-0.1 与 WUSH-54 revision 11 已批准范围；不修改产品实现、Schema、迁移、生成客户端或部署脚本。**不是全部 TC 验收完成，不得据此启动 JARVIS/WUSH-56。**

来源附件 SHA256：`a3f3ba921b1f0edf8e371809c2e82be0149084dc6e3ace1fe906ba9f3c53791b`。实际产品 base：`8305976793097a3d38e3a2f29bc1626abe76130d`；分支 `agent/moss/f8df4ba4dfb0`；Fork `https://github.com/EdenWu3698/multica`。规格中的 InFrame 旧主规范已被本任务明确排除。

## 当前结论

- 真实入口红灯：合同 GET 返回通用文本 404，PUT 返回 404，尚无版本化合同入口。后端套件 57 个叶子用例失败，3 个叶子用例通过；通过项是匿名访问现有认证门、普通任务编辑和普通任务 API 读耗时。
- 真实解析红灯：现有 IssueSchema 的 loose 字段接受非法 `contract_required`（字符串/数字/对象/数组），6 失败；3 个已有兼容行为通过。未改 Schema 使其变绿。
- 运行时既有行为基线：真实 `Daemon.runTask` 面对 8 种 HTTP 拒绝均不启动假业务进程、不改变合成源码摘要、不发送开工后的请求；1 个 HTTP 200 阳性对照确实执行假进程并改变源码，共 9 通过。**这证明收到拒绝后的本地行为，不证明服务端已作合同裁决。**
- 既有后端基线 5 通过；既有前端相关基线 72 通过。它们不等于新合同语义通过。
- Web 入口脚本使用 TestApiClient；仅完成收集检查。无固定 SHA Web 预览交接，未执行浏览器登录、表单或合同交接；无页面性能数据。

红灯不是导入失败或全部 skip。后端首败均先验证现有任务读取成功，再请求新合同入口。**依赖发布成功的断言尚未到达，57 失败不能解释为 57 条内部业务规则已分别被证伪。**

## 复现

环境：macOS arm64、Go 1.26.9（与 go.mod 一致）、Node 23.9.0、pnpm 10.28.2；Go 通过既有 agent CLI guard 运行，仅执行测试创建的假可执行文件；不查找/调用真实智能体。

从仓库根目录安装当前 lockfile 的测试依赖：

```sh
pnpm install --frozen-lockfile --ignore-scripts --filter @multica/core... --filter @multica/views... --filter @multica/web... --filter multica
```

本次使用仓库公共脚本创建 `.env.worktree` 和专用数据库 `multica_wush54_contract_tests_908`（localhost:5432）。只建立测试数据库并运行当前已有迁移，没有启动 Web/API 预览或 daemon：

```sh
WORKTREE_NAME=wush54_contract_tests bash scripts/init-worktree-env.sh
make migrate-up
set -a
. ./.env.worktree
set +a
node tests/task-contract/run.mjs all tests/task-contract/evidence/local-run
```

不要覆盖已有 `.env.worktree`，不要 source 生产配置。runner 的 server 模式要求显式 loopback 数据库及 `multica_wush54_` 前缀；不会使用后端 TestMain 的默认数据库。可用 `W54_GO` 指定本地 Go 路径。runner 不重试、不吞失败，并检查真实用例数量，避免 TestMain 无数据库时退出 0 被误算为通过。预期整套 exit 1（保留真实红灯），不要改断言来取得 exit 0。

其他本轮实际命令：

```sh
node --test tests/task-contract/preflight.test.mjs
node tests/task-contract/generate-fixtures.mjs
pnpm exec playwright test e2e/task-contract.spec.ts --list
# 下列命令在 server/，并继承上述隔离 DATABASE_URL：
bash ../scripts/go-test-with-agent-cli-guard.sh -- go test ./internal/handler -run '^(TestRevisionConflictsPreserveLatestIssueAndComment|TestTextBaselinesIgnoreUnrelatedAggregateRevisionChanges|TestConcurrentRevisionWritesHaveExactlyOneWinner|TestNoOpIssueUpdateDoesNotAdvanceRevisionOrUpdatedAt|TestRerunIssue_PrivateHistoricalAgent)$' -count=1 -json
```

`preflight.mjs` 单独执行时缺预览前提以 exit 2 关闭；这是环境阻塞，不能记作产品红灯。浏览器执行还需 TARS/Data 交接 `preview-manifest.json`（source_sha、synthetic_only、no_real_agents、stop_instructions），专用 `PLAYWRIGHT_BASE_URL` 和 `W54_SYNTHETIC_ONLY=1`。脚本在登录/写入前核验 `/health.commit`。清单字段不是新增产品接口。

## FR / A → FD → SP → 层级

| 验收 | 对应规格 | 本轮实际覆盖 | 未完成部分 |
| --- | --- | --- | --- |
| A-01 | FR/FD/SP-01 | L3/L4 发布、必填缺失/null、空集合入口红灯；合成源码/文档 A/B | L1 领域规则、嵌套字段穷举、固定来源摘要校验 |
| A-02 | FR/FD/SP-02 | L3/L4 版本/历史/并发用例已编写，停在首次发布；已有 revision 并发基线 | 原子持久化、变更记录全文、queued/start 互斥、重启恢复 |
| A-03 | FR/FD/SP-03 | L2 真实 runTask 的拒绝副作用基线；L3/L4 交接入口红灯 | 每个 claim/start/retry/rerun/wakeup/resume 入口，持久拒绝与 receipt fencing |
| A-04 | FR/FD/SP-04 | L4 缺字段/权限/幂等候选测试；已有私有智能体 rerun 基线 | 新合同跨租户/资源逐项读取计数，响应丢失同键确认、去重授权先行 |
| A-05 | FR/FD/SP-05 | L4 普通任务解析基线及 adoption marker 红灯；L5 现有历史按需展示基线 | 当前合同注入/恢复、旧协议/客户端完整旁路矩阵 |
| A-06 | FR/FD/SP-06 | L5 既有冲突对比组件基线；L6 TestApiClient 入口脚本已收集 | 合同 UI 全流程、旧标签页/草稿、页面性能、隔离预览实际执行 |
| A-07 | FR/FD/SP-07 | 生成器、固定数据样本、首败日志、运行脚本、下表追溯 | 完整 TC-01–18 与 N 尚未满足，不能签发正式回归通过 |

L7：本阶段没有真实模型质量目标，也没有真实模型调用授权；未调用模型。确定性服务端门禁与假运行时基线不冒充 L7 Eval。新合同 L1 的产品领域入口尚不存在；没有编写一个测试自有的合同校验器来冒充产品行为。

## TC → 实际测试 → 证据与缺口

表中 Go 名称前缀均为 `TestW54Contract_`，文件为 `server/cmd/server/task_contract_acceptance_test.go`。入口红灯见 `evidence/server-verified`；首败见 `server-first.jsonl`。全部子项保留，不因本轮未覆盖而删除要求。

| TC | 已编写/执行的位置 | 本轮结论与仍欠验证 |
| --- | --- | --- |
| 01 | `TC01_MissingIsMachineReadable`、`TC01_MemberPublishReadAccept` | 缺合同错误码/发布入口红灯；实际 accepted/至多一次及默认上下文尚未到达 |
| 02 | `TC02_RequiredFields`：17 字段 × missing/null | 34 个入口红灯；空白/占位、嵌套字段、服务端绑定字段/并发字段仍待完整穷举 |
| 03 | `TC03_ExplicitEmptyOptionalArrays`、`TC03_RequiredCollectionsCannotBeEmpty` | 1+7 入口红灯；不是数组合法性已被独立执行 |
| 04 | `TC04_BlockingQuestion` | 入口红灯；假设、不可读/失效附件、非阻断问题分支未实现测试 |
| 05 | `TC05_OldVersionAndImmutableHistory` | 入口红灯；新版本/历史/旧确认断言尚不可达；Web 旧标签页未验证 |
| 06 | `TC06_ConcurrentPublish`，已有 `TestConcurrentRevisionWritesHaveExactlyOneWinner` | 新合同入口红灯；channel 屏障已写；既有 revision 并发通过，不等于新合同事务通过 |
| 07 | `TC07_CommentRevisionVersusContent`，已有 `TestTextBaselinesIgnoreUnrelatedAggregateRevisionChanges` | 新合同入口红灯；既有聚合 revision/内容基线通过；被引用评论被编辑分支未写 |
| 08 | `TC08_FloatingCodeBaseline`；`fixtures/manifest.json`、`dirty.patch` | 非固定 SHA 入口红灯；真实 A/B/dirty/D1/D2 已生成；认证运行时观测漂移尚无 wire 适配 |
| 09 | `TC09_UnknownActionAndReceiver` | 2 入口红灯；目录外、路径逃逸、允许/禁止重叠仍欠可运行断言 |
| 10 | `TC10_AnonymousCannotRead`；已有 `TestRerunIssue_PrivateHistoricalAgent` | 现有匿名门/私有目标基线通过；合同跨租户、撤权、伪造回执、禁止越权读取仍未验证 |
| 11 | `TC11_IdempotentMemberHandoff` | 入口红灯；同键/旧版重放断言不可达；双击并发、异载荷、响应丢失未注入 |
| 12 | 尚无产品执行测试 | 发布与 claim/start 的受控事务屏障、旧 queued 原子终结/不 retry 缺已定义的绑定接口；仅书面登记，**不算覆盖** |
| 13 | `TC13_AdoptionCannotBeUnsetByOrdinaryUpdate` | 入口红灯；单/批/legacy claim、重启、rerun/retry/wakeup/resume 全入口尚欠，不能用该测试替代 |
| 14 | `TC14_ActiveRunBlocksPublish`：3 状态 | 3 入口红灯；预置现有任务行不启动进程；取消请求仍活跃/终态后发布及事务屏障尚欠 |
| 15 | `server/internal/daemon/task_contract_acceptance_test.go::TestW54Runtime_TC15_RefusalHasNoLocalExecution` | 8 拒绝+1 阳性对照通过；仅现有运行时拒绝行为；没有注入产品存储/校验故障或响应丢失 |
| 16 | `e2e/task-contract.spec.ts`；上述 L5 组件基线 | TestApiClient 登录/任务/评论/刷新脚本可收集；浏览器未运行，合同表单/交接/异常 UI 尚欠 |
| 17 | `TC17_OrdinaryIssueBaseline`、`TC17_ReadLatencyBaseline`；`packages/core/api/task-contract-acceptance.test.ts` | 普通编辑/API 耗时与 3 解析兼容基线通过；6 非法 adoption 字段红灯；未知准入状态/长历史/旧 runtime 未验证 |
| 18 | `TC18_SeparateRequestsPreserveSnapshot`、preview preflight、合成数据生成器 | snapshot 入口红灯；preflight 2 通过；没有进程重启恢复测试，没有预览验收 |

## 统一负向断言 N 的实际边界

| N 项 | 可执行证据 | 不能据此宣称的内容 |
| --- | --- | --- |
| 不启动模型/业务进程 | runTask 的假 shell 写 marker；200 阳性对照证明能观察执行，8 拒绝 marker 不存在 | 服务端准入是否正确、所有执行入口、真实模型语义 |
| 不写源码 | 同一假 shell 在允许时写合成源码；拒绝前后 SHA256 相同 | 任意已启动进程的一切本地工具副作用 |
| 同步拒绝无运行增量 | HTTP 测试 cleanup 独立读 `agent_task_queue WHERE issue_id` 前后 count | 新合同路由实现后的内部队列/下游语义（当前路由不存在） |
| 无下游运行/自动 retry | 本地拒绝后无后续 HTTP 请求、legacy start 一次；数据库相同 Issue 的总 count 不变 | queued 终结、跨 Issue 下游、服务端自动 retry 的完整断言仍欠 |
| 无越权读取 | 仅既有 private rerun/匿名基线 | 新合同逐资源读取 spy 与泄露审计尚未接入，**未证明为零** |
| 网络结果未知先同键确认 | README 明确约束；候选幂等测试只覆盖明确响应 | 尚无响应丢失注入用例，**不能把未知计为零** |

## 数据与首败纪律

生成器在全新输出目录创建真实本地 Git A/B 和 dirty patch；不复用业务仓库、不设置全局身份。样本含 W1/W2、3 个 W1 成员、W2 外部成员、假智能体/私有目标、当前/旧协议运行时的语义标签、普通/受保护任务 ID、C1、附件、D1/D2、200 条历史和重复事件。`fixtures/manifest.json` 记录实际 UUID/SHA/摘要；这些 UUID 尚未全部 seed 到数据库，不能把数据清单视为已执行覆盖。Go 测试通过既有 `testutil.Fixture` seed；E2E 通过既有 TestApiClient。

首轮 runtime 阳性对照因缺少 AuthToken 失败；第二轮因合成 token 缺少既有 `mat_` 前缀失败。两次均为**测试夹具错误**，日志保留为 `runtime-first.*` / `runtime-final.*`（后者历史文件名不代表最终结论）；修正后以 `runtime-verified` 的 9 通过为准。没有把夹具错误算作功能红灯，也没有降低断言。首次 Playwright 收集缺 dotenv，是过滤安装范围遗漏 web 依赖；补装既有锁定依赖后收集通过，未改 package/lockfile。

## 必须交 Asurada 确认的阻塞项

W54-SPEC-0.1 §1 明说是语义字段而非 canonical JSON Schema，§4 只定义候选路由。为保证后续红灯不是测试自造协议，目前不能完成以下正式断言：

1. **wire/固定摘要**：PUT 完整快照的封装方式、服务端生成字段是否必须出现在请求中；`Source.digest` 对 Issue/Comment 的规范化字节与算法、`source_refs` 的可解析形状。当前 Go builder 是集中适配的候选，并未据 404 证明它是合法生产输入。
2. **准入绑定与持久拒绝读取**：handoff/accept 的响应 envelope、持久拒绝的读入口；认证 runtime 的合同协议声明、SHA/dirty 观测、版本/input_digest/claim_generation 传输绑定。不能用测试自行发明的字段冒充服务端已认定的保护事实。
3. **并发/故障测试接点**：发布/claim/start 的事务屏障与故障注入依赖具体 service/transaction 接点；没有实现接口时，测试自建状态机只能测到自己。本阶段不越权实现产品 service/Schema 以满足测试。

请 Asurada 收敛上述最小接口/测试接点约定（必要时由 Samantha 补规格），再由 MOSS 补齐下表缺口。保留当前文件、提交、标准与首败；不因通用 TDD 要求删除已有实现或弱化断言。当前阶段结论是**部分红灯/基线已交付，完整正式测试包阻塞**；不是实现、回归、独立复核、集成、部署或最终验收通过。
