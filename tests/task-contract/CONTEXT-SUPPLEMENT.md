# WUSH-54：补齐启动上下文、实际代码观测与跨进程 N 测试

2026-10-30 接续 `c0a34bffb282bfddbed76f71c593d74e807cce1c`；需求仍为原规格及已核验 wire 补充，权限、产品 base 与原远端分支不变。上一轮结束时没有后台工作，`in_progress` 表示工作尚未完成。本轮继续本人可独立完成的测试，不将“已交部分红灯”当整阶段完成。

## 新增 16 项及证据边界

| 追溯 | 正式测试位置/名称 | 断言与首测 |
| --- | --- | --- |
| FR/A-05 → FD-05 → SP-05 → TC-13/17，WS-03 | daemon `task_contract_context_test.go` / `CurrentContextAtProcessBoundary`，fresh/resumed/large_current_contract（3 项） | 捕获真实假进程收到的 stdin、argv、CLAUDE.md/AGENTS.md；当前目标、禁止、验收、责任人、全部停止条件、handoff/digest 必须到达。约 128 KiB 当前验收内容要完整提供或明确拒绝并不启动。三项均启动一次却缺当前合同内容，真实红灯。resumed 表示携带旧会话的运行请求，不冒报真实模型会话恢复成功。 |
| FR/A-04/05/07 → FD-04/05/07 → SP-04/05/07 → TC-08/15/17，WS-03/04 | 同文件 / `ActualGitObservationBeforeExecution`，clean accepted/rejected、SHA drift、dirty 无固定证据、dirty fixed accepted/rejected（6 项） | 实际创建 Git A/B/dirty，固定合成提交时钟；真实 local_directory 原位环境；Git wrapper 透传实际 Git 并计数，hooks/源码哈希/上传/进程均有探针。六项实际 start 都缺 code_observations，是真红灯。4 个拒绝对照未执行业务进程、源码不变；2 个接受对照确实启动。没有证明观测已采集或 dirty 附件已核验。 |
| FR/A-03/04/07 → FD-03/04/07 → SP-03/04/07 → TC-01/04/15，WS-03/04 | server `task_contract_process_test.go` / `AppAPIToActualProcess`（4 项）；daemon `task_contract_bridge_test.go` | 真 App API/PG 产生 claim → 独立测试子进程解码 → 真 runTask/start → 假业务可执行文件；真实资源读取 spy、工作区运行计数、任务状态、源码哈希及持久拒绝在同一场景观察。普通任务对照通过；受保护对照、claim 后资源变更/删除三项止于 PUT 404，不能宣称 N 已贯通。 |
| FR/A-05 → FD-05 → SP-05 → TC-13/17，WS-03 | 同 server 文件 / `DefaultClaimOmitsHistory`（3 项） | 实际存入 12 条大评论与历史任务结果，分别普通任务、受保护 fresh/resumed，检查默认 claim 不含完整旧正文，受保护完整 v2 不被替换。普通路径通过，受保护两项止于 PUT 404。 |

新测试没有删除旧断言；此前 477 项保留。最终固定结果见 `CONTEXT-RESULT.md`，逐例 source SHA、首败和分类见 `evidence/context-fixed/leaf-results.json`。`context-first` 是运行时首败；`context-server-first` 误把 Storage.Delete 当返回 error，`context-server-second` 误用不存在的 output 列，原日志保留。按实际 Storage API 和 agent_task_queue.result JSON 修正，仅属于夹具错误，不计产品缺陷；`context-server-third` 得到真实路由首败及两个阳性基线。

## 子进程与隔离

daemon 的测试 TestMain 在隔离用户目录后检查专用测试输入变量；只测试二进制包含该分支，没有产品符号或公共 API。输入要求 loopback App API，所有探针文件位于合成根目录；真实测试 token 仅进 0600 输入文件，不进命令参数和日志。测试子进程使用测试创建的固定可执行文件，沿用外层 agent CLI guard，并同步等待退出。普通控制证明跨包接线实际工作，不能用它替代受保护断言。

本轮仍无产品 hook、事务、Schema、迁移、生成客户端或部署改动。浏览器完整操作与取证步骤已列在 `WEB-ACCEPTANCE.md`，它是待执行脚本；现有自动入口两项继续保留。没有虚构不存在的表单控件，也不将手工步骤算自动用例数。

## 后续真实依赖

- 产品合同路由及 WS-05 语义接点尚无实现，合同链路的深层断言与并发故障仍未到达；hook 命中 0 必须如实保留。
- document 固定 Git blob 的服务端权限/读取适配仍无产品接点；需要后续实现者提供实际适配边界，MOSS 才能验证读计数、symlink 不跟随与固定 blob 权限，不能自建替代校验服务。
- TARS/Data 的固定 SHA 隔离预览尚缺；完整 Web 表单/旧标签页/异常流程的真实控件绑定和浏览器执行仍未完成。

这些是明确的剩余接入与验证项，不是规格被撤回，也不把任务整项标成 blocked。交付交 Asurada 按原阶段标准核验；MOSS 不自行启动 JARVIS、WUSH-56 或部署。
