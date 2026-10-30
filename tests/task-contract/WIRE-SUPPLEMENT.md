最新接续：2026-10-30 的启动上下文、实际 Git 观测与跨进程 N 测试见 [CONTEXT-SUPPLEMENT.md](CONTEXT-SUPPLEMENT.md)，固定实跑见 [CONTEXT-RESULT.md](CONTEXT-RESULT.md)。以下保留前轮记录。

# WUSH-54：已核验 wire 补充的正式测试接续

依据 WUSH-54 revision 25 继续 MOSS 正式测试工作。产品 base 为 `8305976793097a3d38e3a2f29bc1626abe76130d`；接续 head 为 `bb1b66494a2606b44f4b67930efb44e558e5db4a`。实际本地分支为平台创建的 `agent/moss/c431a63450dd`，交付仍快进至原远端 `agent/moss/f8df4ba4dfb0`，不操作旧 checkout。

本轮仅修改测试、测试设施、数据、脚本及证据；未修改业务实现、canonical contract、迁移、生成客户端或部署脚本。没有启动 JARVIS/WUSH-56。本文件是测试交接，不是实现、正式回归或最终验收通过。

## 固定需求和执行结果

原附件与补充原文收在 `specs/`，原文的候选状态是历史文字；本次授权与核验以当前任务正文为准：

- W54-SPEC-0.1：`a3f3ba921b1f0edf8e371809c2e82be0149084dc6e3ace1fe906ba9f3c53791b`。
- W54-SPEC-0.1-wire-supplement：`59190fd9a3f1b54e108d3437b28c32c4118192d43f6d38263cc9f84b5a1de71c`。
- 6 个字节摘要向量、示例 input_digest 和附件指纹共 8 项测试设施校验通过；不计作产品行为通过。正式 API 回执预期按实际 seed 的 contract ID、Issue 内容、revision 和 Source 重算。

固定 SHA 的最终实跑以 `evidence/wire-final/summary.json`、`leaf-results.json` 及 `WIRE-RESULT.md` 为准；首轮日志独立保留。源码提交前实跑如下：

| 套件 | 通过 | 失败 | 跳过 | 分类 |
| --- | ---: | ---: | ---: | --- |
| server | 11 | 222 | 0 | 217 项合同路由缺失；5 项明确的产品接点映射待实现 |
| runtime | 19 | 26 | 0 | 原 18 项基线、1 新 generation 对照通过；19 握手、5 传输能力、2 重复派发红灯 |
| recovery | 10 | 0 | 0 | 既有 generation、丢响应、取消与并发基线 |
| views | 179 | 1 | 0 | 原合同编辑入口红灯保留 |
| core | 3 | 6 | 0 | 首轮非法 adoption 类型红灯复跑保留 |
| 合计 | 222 | 255 | 0 | 477 项；其中 250 项产品红灯、5 项局部待实现，不能统称 255 项业务缺陷 |

此前 96 项红灯和 core 6 项历史红灯均保留。新增 158 个叶子用例；core 9 项只是重新纳入本轮，不能算新增覆盖。API 的 217 个失败大多在首次 PUT 404，后续逻辑并未各自被证伪。5 个 LOCAL_PENDING 是 WS-05 映射缺失及首次发布故障接入，命中数 0；不是导入错误、事务通过或存储回滚已验证。

新增 runtime 19 项运行真实 `Daemon.runTask`：合法回执阳性对照的假进程确实执行一次，但实际 start 缺 `task-contract-v1`/contract_admission，因此仍失败；其他 18 个非法/缺失/未知/错绑定 HTTP 200 均实际启动假进程并改写合成源码。它们证明当前客户端未实现受保护握手，不证明服务端已作合同裁决。没有调用真实智能体或模型。

## WS → FR/A → FD → SP → 测试

S=`server/cmd/server/`，D=`server/internal/daemon/`。表中函数前缀 `TestW54Contract_` 省略；runtime 前缀 `TestW54Runtime_`。逐叶子实际命令、SHA、结论、首次诊断见 `leaf-results.json` 和同目录原始日志。

| 补充 | 原追溯 | 新增测试位置 | 实际边界 |
| --- | --- | --- | --- |
| WS-01 | FR/A-01/02/04 → FD-01/02/04 → SP-01/02/04 | S `task_contract_wire_test.go`：完整 PUT/GET/no-op/CAS/change、只读/未知字段、重复 JSON key、完整 Source、引用矛盾、评论原字节/编辑 revision、非法路径；S `task_contract_sources_test.go`：真实附件/代码语料 | 合同入口缺失；结构/摘要/版本业务断言未到达 |
| WS-02 | FR/A-03/04/06 → FD-03/04/06 → SP-03/04/06 | wire：成员 pending/accept/receipt、同键持久拒绝、handoff/admission 单项/列表/过滤、撤权；S `task_contract_restart_test.go`：新 OS 进程/新连接读取当前、历史、回执和拒绝 | 普通 Issue 独立进程重读基线通过；完整合同持久性用例停在 PUT |
| WS-03 | FR/A-03/05 → FD-03/05 → SP-03/05 | S `task_contract_binding_test.go`：单/批/legacy claim、持久及连接能力、认证 mdt、member/mat/其他 runtime、版本/digest/plan/generation、恢复入口；sources：实际 Git A/B/dirty；D `task_contract_wire_test.go`：真实 start 请求与 200 回执失败关闭；D `task_contract_transport_test.go`：实际注册/HTTP/WS 声明及真实 poller 重复派发 | mdt 普通路径阳性对照通过；合同服务端入口缺失；运行时 26 项真实红灯 |
| WS-04 | FR/A-03/04/07 → FD-03/04/07 → SP-03/04/07 | wire：精确 RFC6901 Pointer/稳定码、权限撤回及只读；sources：包装真实 LocalStorage.GetReader 的计数/故障；既有 N 加工作区队列/评论/inbox/assignee/status 观察 | 现有附件读取合法计数 1、跨工作区计数 0；合同路径仍未到达，不冒报完整 N |
| WS-05 | FR/A-02/03/07 → FD-02/03/07 → SP-02/03/07 | S `task_contract_atomicity_test.go`：受控两写者、发布/claim 顺序、三个 before_commit、start 源变化重查、取消、逐持久步骤故障、丢响应/结果未知后原键确认 | 只有测试侧 channel/计数/清理与真实 API/DB 断言；产品映射为空，全部语义 hook 命中 0，未验证事务 |

## TC 接续与仍未验证项

| TC | 接续测试/证据 | 未到达或尚未覆盖 |
| --- | --- | --- |
| 01 | WS01 CompleteSnapshotNoOpCASAndChange；WS02 ReceiptAndDurableRead；WS03 ClaimStartAndReceipt | API 入口红灯；正常服务端闭环未运行 |
| 02 | 保留原必填矩阵；WS01 ReadOnlyAndUnknownFields、DuplicateJSONKey、SourceShapeAndConflict | 后续精确字段错误未到达；未以 404 当字段校验已执行 |
| 03 | 原显式空数组/必需集合断言全部保留 | 合法快照发布未到达 |
| 04 | 原阻断问题/假设；WS01 ArtifactAvailabilityAndBytes 的 valid/digest/deleted/cross-workspace/storage-fault | 合同读取与注入点未到达；附件下载 spy 的既有控制通过不替代合同校验 |
| 05 | 原历史/旧键重放；WS02 RestartPreservesAllContractFacts | v2 与旧回执历史重读未到达；旧浏览器页仍待预览 |
| 06 | 原并发测试保留；WS05 TwoWritersAtProductArbitration | ready/release 对应真实仲裁接点未接，不能用请求同时发出证明锁竞争 |
| 07 | WS01 CommentBytesAndRevision；WS05 SourceRecheckedAfterClaim | 来源内容/精确 revision/claim 后复验未到达；原普通评论聚合 revision 基线保留 |
| 08 | WS03 StartBindingMutations、CodeObservationMatchesActualSeed；固定 A/B/patch、6 字节向量 | SHA 实际生成/读取；来源绑定服务端断言未到达，文档仓库权限/固定 blob 读取及 runtime 元数据采集仍未端到端验证 |
| 09 | 原动作/目标/目录/禁止优先；WS01 InvalidRepositoryPaths | 保留全部范围断言；路径格式已适配，后续裁决未到达 |
| 10 | WS03 MemberCannotSubmitRuntimeObservation；WS02 AcceptRejectsForgedBinding；WS04 ReadProjectionRevocation/ResourceReadSpyBaseline | 普通 mdt/附件权限阳性基线通过；合同权限/所有不可见字段仍未验证 |
| 11 | 原同键/异载荷/撤权优先；WS02 RejectionBeforeHandoffIsQueryableAndReplayStable；WS05 同键故障恢复 | 去重及拒绝持久断言在缺入口之后；同键真实并发的产品竞争仍未命中 |
| 12 | WS03 OldGenerationCannotCancelReclaimedRun；WS05 PublishAndClaimOrder/UncommittedPublishIsInvisible/ClaimCommitProtectsActiveVersion | 实际 generation 取真实 claim；后续原 ID cancelled/receipt 不覆盖/无 retry 未到达 |
| 13 | 单/批/legacy；issue_rerun、task_rerun、真实 MaybeRetryFailedTask、真实 wakeup trigger、旧会话 claim；runtime resumed 分支 | 恢复测试先建立有效 receipt，现停在发布；实际 WS 建连及 HTTP 单/批/legacy/注册均缺能力；真实调度重放重复执行，同批重复有多余失败回报；新 generation 对照通过。服务端 WS 合同裁决仍未验证 |
| 14 | 原三活跃/三终态；WS05 StartCommitAndCancellation | start/发布互斥与“请求取消≠终态”尚未到达 |
| 15 | 所有规定发布/交接/拒绝/接手写入故障；operation.after_commit_before_response、commit.outcome_unknown × publish/handoff/start | 未命中注入，不声称回滚/提交唯一；未知结果始终按原版本/键/generation 确认 |
| 16 | 原组件入口红灯；原 E2E 两条重新收集 | 无预览，浏览器未执行；完整表单/旧标签页/错误操作脚本仍须真实 UI 交接后补齐，不把入口脚本冒充全流程 |
| 17 | core 3/6；views 179/1；普通 claim/API 性能；runtime 未知状态/错误绑定 200 真红灯 | Web 未知状态交互和完整上下文注入未验证；普通路径通过不代表受保护能力通过 |
| 18 | 新 OS 进程重建真实 router/独立 PG pool 的普通读取通过；完整合同七个投影重读已写 | 合同恢复未到达；固定 SHA 隔离预览仍待 TARS/Data |

## 接点、N 和角色边界

`w54AttachProductSeams` 是测试侧待接适配函数，不是产品 hook 实现、公共入口或第二套业务服务。后续获授权的 JARVIS 提供真实内部接点；MOSS 将它们映射到 WS-05 名称后正式实跑。不得为了让测试通过向它注册假合同事务。每个屏障有 ready/release，超时只作失败保护，清理释放屏障、取消并收回 HTTP worker。未知接点配置立即报错的测试设施断言已通过。

三类观察均有源码：真实 App API/隔离 DB、真实 runTask 假进程与源码 SHA、真实 LocalStorage 读取 spy。它们目前分套件运行，尚未证明从合同服务端拒绝到本地停止的完整贯通 N。GET/发布/同步拒绝的队列、assignee/status、评论和 inbox 观察是实际 DB 读取；没有声称拦截任意工具副作用或验证全部通知通道。

L2 运行时、L3 真实 PG/对象文件与独立进程、L4 wire、保留 L5 基线；L6 仅收集，L7 未执行且不在本阶段模型质量范围。无 L1 合同领域入口，因此不编写自有业务校验器冒充产品测试。

## 复现与首败

macOS arm64；Go 1.26.9；pnpm 10.28.2。新运行目录通过公共脚本生成 `.env.worktree`，独立数据库 `multica_wush54_wire_supplement_839`，localhost:5432，使用既有迁移至 594。仅复用前轮安装的 Go 编译器，不移动旧工作区/成果/环境配置；不切换现有 daemon。

```sh
# 本 checkout 已存在配置，勿覆盖；新 checkout 才运行下面两条
WORKTREE_NAME=wush54_wire_supplement bash scripts/init-worktree-env.sh
make migrate-up
# 导入本地专用测试数据库配置
set -a
. ./.env.worktree
set +a
# 若 go 不在 PATH，W54_GO 指向 Go 1.26.9 二进制
node tests/task-contract/run.mjs all tests/task-contract/evidence/wire-final
node tests/task-contract/report-wire.mjs tests/task-contract/evidence/wire-final
node --test tests/task-contract/verify-wire-vectors.mjs tests/task-contract/preflight.test.mjs
pnpm exec playwright test e2e/task-contract.spec.ts --list
```

runner 预期 exit 1；没有 skip，不吞失败，不把零用例算通过。日志中 go 工具路径和 Vitest 输出路径只是复现上下文，交付通过 issue 附件/远端源码，不用本地路径冒充交付链接。

首轮 `wire-first` 有三处将 `testutil.Row.Scan` 当作返回 error 的编译错误；按既有 Row API 修正，原日志保留，不算产品红灯。运行时响应夹具初版 changed_fields 在源码自查时由仅 /goal 补为完整初建业务字段列表；不改任何行为期望，04ed7271c 的固定实跑保留在 wire-fixed-04ed7271c。根目录 ESLint 没有配置，`wire-lint.txt` 的 exit 2 是命令不适用；未改项目配置凑通过，JS 使用 node 语法检查及 8 个向量实跑。Go 完整编译/选定测试通过收集；gofmt、diff 检查作为交付门。浏览器 preflight exit 2 仍是缺固定预览前提，不能算功能红灯。

本轮需求级 wire 阻塞已解除。缺产品入口/接点继续作为真实红灯或局部待实现；任务保持 in_progress，不能因本轮运行结束或部分通过声称完整阶段验收。下一步由 Asurada 核验本测试交付及剩余接入顺序，未启动实现、其他包或部署。

## 交付核对追加：实际传输与重复派发

`task_contract_transport_test.go` 新增 8 项：5 个真实客户端传输缺能力红灯；单 slot 重放同一 task/generation 实际执行假进程 2 次并上报 complete 2 次；同批重复的并发样本执行 1 次但额外上报 fail 1 次；不同 generation 的阳性对照执行/complete 各 2 次。后两项重复派发断言要求恰好一次执行且无额外终态回报，不把环境锁拒绝当业务去重。它们运行真实 runBatchPoller→handleTask→runTask，仅服务端响应与业务可执行文件为固定合成夹具。

传输首轮的三个调度超时源于 New 默认用测试二进制启动 CLI 环境 helper；首败 `wire-transport-first` 与诊断日志保留。改用现有测试支持的真实进程内 execenv.Prepare 后得到 `wire-transport-second` 的产品行为红灯，不把原超时计入产品结论。WS 连接调用等待全部读写/心跳退出；poller 停止后显式等待所有 task worker。没有新增业务实现。
