# WUSH-54 2026-10-30 固定测试结果

实际测试源码：`df190d50f38e87cd0779c23670dd45d24a3ebe6c`。续测起点：`c0a34bffb282bfddbed76f71c593d74e807cce1c`；产品 base：`8305976793097a3d38e3a2f29bc1626abe76130d`。仓库 `https://github.com/EdenWu3698/multica`；本地分支 `agent/moss/c431a63450dd`，原远端交付分支 `agent/moss/f8df4ba4dfb0`。最终证据提交 head 在任务回帖与附件封面列出；与测试源码间只有证据和报告。

**493 项：224 通过、269 失败、0 跳过。** 失败分为 264 项行为红灯、5 项产品语义接点待接入。222 项红灯止于缺失合同路由，后续断言没有执行，不能把每项都说成独立定位的业务缺陷。前轮 477 项完整保留；本轮新增 16 项，2 通过、14 失败。

| 套件 | 通过 | 失败 | 跳过 | 本次新增 |
| --- | ---: | ---: | ---: | --- |
| core | 3 | 6 | 0 | 0；历史 adoption 类型红灯保留 |
| runtime | 19 | 35 | 0 | 9 个真实红灯：启动输入缺合同 3，Git 观测缺失 6 |
| server | 13 | 227 | 0 | 2 个阳性基线，5 个缺合同路由红灯 |
| recovery | 10 | 0 | 0 | 0；恢复基线保留 |
| views | 179 | 1 | 0 | 0；合同编辑入口仍缺失 |
| 测试设施（另计） | 10 | 0 | 0 | 指纹/摘要向量及隔离 preflight；不算产品行为通过 |

实际 `runTask` 启动的假进程在 fresh、携带旧会话、约 128 KiB 验收内容三种场景中都没有收到完整当前合同。实际 local_directory 的六种 Git 场景中，发出的 start 均缺少 code_observations；4 个 409 拒绝对照未启动假进程且源码不变，2 个接受对照实际启动。Git 命令、hooks、源码哈希与上传探针没有发现本轮未经准入的业务源码写入/上传；这些局部对照不代表元数据功能已实现。

新增跨进程测试已将真实 App API/隔离 PG → 真 claim → 独立 daemon 测试子进程 → 真 start/runTask → 假业务进程接在一起。普通控制实跑 executions=1、数据库 running、文件摘要变化，证明接线有效。受保护控制及 claim 后附件变化/删除都止于 PUT 404；N 的数据库/资源读取/源码/进程与持久拒绝断言已写入同一场景，尚未到达，不能宣称保护链路已验证。大历史默认 claim 的普通控制通过，两个受保护 v2 场景仍止于 PUT 404。

命令和原始日志：

```sh
set -a
. ./.env.worktree
set +a
# 非 PATH 安装使用 W54_GO 指向本机已安装的 Go 1.26.9
node tests/task-contract/run.mjs all tests/task-contract/evidence/context-fixed
node tests/task-contract/report-wire.mjs tests/task-contract/evidence/context-fixed
node --test tests/task-contract/verify-wire-vectors.mjs tests/task-contract/preflight.test.mjs
pnpm exec playwright test e2e/task-contract.spec.ts --list
```

runner exit 1 符合当前红灯；report exit 0 表示 493 个叶子全部分类完成，不表示产品通过。逐 TC/WS、实际 source SHA、首败诊断见 `evidence/context-fixed/leaf-results.json`。编译误用 Delete 返回值和历史任务 output 列的夹具错误已修正，首败目录仍保留，未计为产品红灯。gofmt、改动的 whitespace 检查、report JS 语法检查通过；本轮不宣称仓库 lint/typecheck 全绿。

性能只是普通任务读取基线（1 合成 Issue，1 预热 + 30 样本，毫秒）：详情 p50=1.337、p95=1.506、max=1.558；列表 p50=1.048、p95=1.136、max=1.325。没有合同成功路径或页面性能结果，不作提速/无退化结论。

环境沿用本 checkout 专用 `multica_wush54_wire_supplement_839` 数据库与原有 594 迁移，macOS arm64、Go 1.26.9、Node 23.9.0、pnpm 10.28.2。本轮测试子进程均已收回，启动的 PostgreSQL 容器已停止；没有切换当前 daemon、连接生产数据或调用真实智能体/模型。

浏览器仅收集现有 2 项；未运行浏览器。完整 12 项 Web 操作与取证单在 `WEB-ACCEPTANCE.md`，不把这份待执行单计为自动用例或通过。固定预览 manifest、实际合同表单仍缺；服务端 document 固定 blob 权限读取的实际适配、WS 合同裁决及 WS-05 产品 hook 也仍待接入。语义 hook 命中 0，完整事务/拒绝 N/浏览器验收均未通过。

任务保持 `in_progress`，本轮只交付已完成的测试增量与真实结果。下一依赖明确为实现方的真实入口/接点及 TARS/Data 的固定隔离预览；接入顺序由 Asurada 按既有范围核验。本轮没有启动 JARVIS/WUSH-56、产品实现或部署，不将局部缺实现升级为规格阻塞，也不冒报完整阶段验收。
