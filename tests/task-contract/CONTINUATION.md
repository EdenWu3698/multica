> 本轮 wire 补充与最新证据见 [WIRE-SUPPLEMENT.md](WIRE-SUPPLEMENT.md)；以下保留历史结论，不再作为接口规格阻塞依据。

# WUSH-54 续测交接

本轮依据 WUSH-54 revision 20、W54-SPEC-0.1 和上轮 `d2038e096eff7cf4963e1ba7912efa6a4d3891cc` 继续。原产品 base 为 `8305976793097a3d38e3a2f29bc1626abe76130d`。不改产品、Schema、迁移、生成客户端或部署脚本，不启动 JARVIS/WUSH-56。**整项保持 in_progress；局部待接入不等于整项 blocked。**

平台在本轮运行目录自动创建实际工作分支 `agent/moss/3deace8afab7`，起点精确为上轮 head；沿同一测试体系续写。旧工作区和旧本地分支未修改。获授权的交付目标仍为 Fork 的 `agent/moss/f8df4ba4dfb0`，仅允许快进推送。最终 head 和固定 SHA 重跑证据见 issue 的本轮附件。

## 实测与首败

| 本轮套件 | 通过 | 失败 | 跳过 | 结论 |
| --- | ---: | ---: | ---: | --- |
| server（原套件与新增 42 项） | 7 | 95 | 0 | 95 项均为合同入口缺失，发布后的业务断言未到达 |
| runtime | 18 | 0 | 0 | 拒绝/恢复/响应丢失的本地行为基线；包括有效假进程阳性对照 |
| recovery | 10 | 0 | 0 | 既有 start 丢响应重放、generation fencing、并发、取消与精度基线 |
| views | 179 | 1 | 0 | 合同编辑入口缺失是真实红灯；包含 200 条历史默认折叠基线 |
| 合计 | 214 | 96 | 0 | 310 个叶子用例；不是全部合同回归通过 |

首轮 core 的 3 通过 / 6 非法类型红灯仍保留，没有改 Schema 或重复执行未受影响的 core 套件，也不把其历史结果混入本轮数量。原 57 项后端入口红灯全部保留；本轮新增 38 个入口红灯、4 个既有行为通过。views 新纳入已有 IssueDetail 套件，是 106 个既有测试和 1 个新入口红灯，不是新增 107 项合同能力。

运行时扩大到旧会话、显式 claim generation、401/404/422、未知非成功响应、断开 start 响应。对 claim 模式读取实际 HTTP body 验证 runtime_id/dispatched_at。拒绝前后的非 start 请求均计数，假进程 marker 不存在、合成源码 SHA256 不变；200 阳性对照执行并改写文件。**响应丢失的远端 accepted 结果为 UNKNOWN**，本轮只证明本地未执行，未据此声称远端零副作用或同键确认已实现。

首轮 UI 探索有两个失败：合同入口缺失是真实红灯；不完整 source_context 夹具导致渲染失败是测试错误。后者按 systematic-debugging 定位后改用 SourceContextBadge 既有完整夹具，搬到相邻规范测试中验证 200 条历史；初始日志仍保留为 `continuation-ui-harness-first.json`。过滤收集产生的 106 deselected 不算运行/跳过验收；随后完整 views 实跑为 0 跳过。

## 逐项追溯与局部待接入

位置缩写：S=`server/cmd/server/task_contract_acceptance_test.go`；C=`server/cmd/server/task_contract_continuation_test.go`；D=`server/internal/daemon/task_contract_acceptance_test.go`。S/C 的测试前缀为 `TestW54Contract_`。首败、命令、退出码与叶子结果保留在 `evidence/continuation-*`。固定 SHA 最终结果随 issue 附件交付。

| TC | 本轮新增/加强的测试位置与结果 | 未到达或待接入子项 |
| --- | --- | --- |
| 01 | S `TC01_*` 保留入口红灯 | WIRE：合法快照/回执 envelope；accepted/唯一运行/最小上下文未到达 |
| 02 | C `TC02_BlankAndNestedRequirements` 16 变体、`TC02_PublishRequiresBothConcurrencyPreconditions` 2 变体，入口红灯 | WIRE：精确字段路径、只读绑定字段、来源编码与 source_refs 规则待补；不能把 404 当嵌套校验已执行 |
| 03 | S 两组测试保留入口红灯 | WIRE：显式空数组/代码不适用的合法发布尚未到达 |
| 04 | C `TC04_BlockingAndNonblockingAssumptions` 2 项入口红灯 | WIRE：附件可读/失效/无权、非阻断 question 与持久拒绝读取；假设分支在发布后未到达 |
| 05 | S 旧版/历史测试保留 | WIRE+PREVIEW：历史及旧确认重放断言未到达；旧标签页保留草稿未验证 |
| 06 | S 受控双发布测试保留 | WIRE+HOOK：版本/变更记录唯一性未到达；事务裁决/故障接点待交接，不用 sleep 造并发证据 |
| 07 | S 聚合 revision 与正文变化测试保留 | WIRE：被引用评论编辑、来源规范化字节；不得复用旧摘要猜算法 |
| 08 | S 浮动 SHA 红灯与原合成 A/B/dirty/D1/D2 保留 | RUNTIME+WIRE：认证代码观测、dirty 附件绑定/文档摘要漂移未接入 |
| 09 | C `TC09_DeclaredResourceBoundary` 6 变体入口红灯；C `TC02_*` 补未知 operation/引用 | WIRE：repository locator 仍集中使用候选适配，须核验后才能认为合法 setup；目录外/逃逸/禁止优先均未到达，不宣称已独立证伪 |
| 10 | C `TC10_CrossWorkspaceReadBaseline` 通过：W2 本人资源阳性读取，W1 目标/伪造 workspace header 拒绝且不泄露合成正文/责任人；撤权、冒签接收人、撤权先于幂等查询 3 项入口红灯 | WIRE+RUNTIME+HOOK：新合同逐资源 read spy、私有 agent、agent receipt 防伪和持久拒绝；普通 Issue 权限基线不等于合同权限通过 |
| 11 | C `TC11_SameKeyDifferentPayload` 入口红灯；既有 start 同 generation 丢响应/并发重放基线通过 | WIRE+HOOK：合同 handoff 双击并发、响应丢失后同键确认；不能用 start 基线替代 handoff 去重 |
| 12 | 原首败和场景保留，尚无合同事务可执行接点 | RUNTIME+HOOK：queued 原子失效、pending stale、publish 与 claim/start 双裁决、无自动 retry；**未覆盖，不记通过** |
| 13 | C `TC13_ClaimEntrypointsRequireCurrentAdmission` 单/批/legacy 三个普通阳性对照通过，三个 adopted 测试停在发布；C `TC13_LegacyStartCannotBypassAdoption` 两个入口红灯；D 的旧会话/claim 拒绝通过 | RUNTIME+WIRE：rerun、自动 retry、wakeup、重启后 adoption、旧 accepted 不复用仍待绑定接口；普通三入口通过不是受保护任务准入通过 |
| 14 | C `TC14_TerminalRunAllowsExplicitNewVersion` 3 终态入口红灯；原三活跃态断言保留 | HOOK+WIRE：只请求取消而未确认停止仍拒绝发布、受控并发；现有 start 取消裁决基线不替代合同发布裁决 |
| 15 | D 拒绝、旧会话、断响应与完整本地探针通过；`daemon_start_claim_test.go` 的丢响应重放基线通过 | HOOK+WIRE：产品发布中途存储失败/回滚、持久拒绝恢复及 handoff 同键确认；远端未知不记零 |
| 16 | `issue-detail.test.tsx` 中 W54 入口红灯；`e2e/task-contract.spec.ts` 两条 TestApiClient 场景收集成功 | PREVIEW+WIRE：浏览器登录/任务/评论/合同表单/交接/刷新/旧页/错误操作未执行；没有固定 SHA 预览交接 |
| 17 | `source-context-viewer.test.tsx` 中 W54 200 条历史默认折叠通过；普通三个 claim 对照、编辑和 API 性能通过 | WIRE+RUNTIME：未知准入枚举失败关闭、默认 claim payload 不携带完整合同历史、受保护旧客户端/旧 runtime 降级门未验证 |
| 18 | 预览隔离 preflight 2 项通过；实际预览 preflight exit 2；S 独立请求重读入口红灯 | PREVIEW+WIRE：进程重启后的当前/历史/拒绝/回执持久性，浏览器与页面性能未执行 |

WIRE、RUNTIME、HOOK 对应 WUSH-59 三组需求补充，由 Samantha 交付、Asurada 核验并回交；MOSS 不自行改标准，也不造产品 service/hook。PREVIEW 对应原规格 §6 已定义的后续 TARS/Data 固定 SHA 隔离预览交接。它们是逐项待接入，当前结果不是整项阻塞。

## N、层级与测量边界

- S/C 的 `noNewRuns` 从单 Issue 扩为整个隔离工作区的 queue 计数，覆盖跨 Issue/无 Issue 的同工作区下游。未覆盖跨工作区副作用、队列取消后自动 retry 和产品内部资源 read spy。
- D 验证真实 runTask 的进程、文件摘要及全部非 start HTTP 请求。只替换远端和可执行文件，不调用用户安装的智能体。服务端是否正确裁决合同仍未实现。
- L2：runtime；L3/L4：真实 PostgreSQL 与 App API；L5：真实共享视图及现有 API doubles；L6：仅可收集，未运行浏览器。L1 无产品合同领域入口，未写测试私有校验器；L7 本范围无模型质量目标/真实调用授权，未执行。
- 当前性能样本：同 checkout / 一个合成 Issue / 1 预热 + 30 样本；普通详情 p50 1.393 ms、p95 1.754 ms、max 1.897 ms；列表 p50 1.002 ms、p95 1.090 ms、max 1.094 ms。来自 `continuation-server-first/server.stdout`。无新增合同成功读取/准入/页面耗时；不作无退化或提速结论。固定 SHA 重跑如有另一次采样，作为独立样本保留，不能择优替换首测。

## 复现与环境

macOS arm64、Node 23.9.0、pnpm 10.28.2、Go 1.26.9。当前 checkout 公共脚本创建独立数据库 `multica_wush54_contract_continuation_680`，localhost:5432；运行既有迁移至 594。未修改或切换 `desktop-api.multica.ai` daemon，未启动 Web/API 持久服务或真实 agent。

```sh
# 仅在没有本 checkout 配置时执行；不复制旧环境，不覆盖已有文件
WORKTREE_NAME=wush54_contract_continuation bash scripts/init-worktree-env.sh
make migrate-up
set -a
. ./.env.worktree
set +a
# W54_GO 可指定 Go 1.26.9 二进制
node tests/task-contract/run.mjs server tests/task-contract/evidence/server-local
node tests/task-contract/run.mjs runtime tests/task-contract/evidence/runtime-local
node tests/task-contract/run.mjs recovery tests/task-contract/evidence/recovery-local
node tests/task-contract/run.mjs views tests/task-contract/evidence/views-local
pnpm exec playwright test e2e/task-contract.spec.ts --list
```

server/views 预期 exit 1；runtime/recovery 预期 exit 0。runner 检查隔离 DB 和非零真实结果、记录工作树状态，不把 TestMain 无数据库时 exit 0 当绿灯。当前预览 preflight 的 exit 2 是缺固定 SHA 预览及专用 Web URL 的环境前提，不是产品红灯。正式结论仍为**红灯与既有行为基线续测交付**，不等于实现、回归、独立复核、集成、部署或最终验收。
