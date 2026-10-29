# WUSH-54 wire 接续固定版本结果

2026-10-29，MOSS。测试源码 SHA：`37204f34a84b8527a9efdee731251f53a6f4cb12`。产品 base：`8305976793097a3d38e3a2f29bc1626abe76130d`；接续起点：`bb1b66494a2606b44f4b67930efb44e558e5db4a`。仓库 `https://github.com/EdenWu3698/multica`；本地分支 `agent/moss/c431a63450dd`，交付远端分支 `agent/moss/f8df4ba4dfb0`。最后证据提交的 head 在 issue 最终评论列出；与实跑 SHA 之间只增加报告和运行证据。

**477 项实际执行：222 通过、255 失败、0 跳过。** 失败分为 250 项产品红灯及 5 项接点待实现，不是正式回归通过。原 96 项红灯、core 历史 6 项红灯、18 项运行时及 10 项恢复基线均保留。新增 158 个用例；core 9 项是历史用例复跑，不能当新增。

| 范围 | 通过 / 失败 / 跳过 | 本次实际结论 |
| --- | --- | --- |
| server | 11 / 222 / 0 | 217 合同路由缺失；5 LOCAL_PENDING（实际 hook 映射及首次发布故障）。产品仲裁接点命中 0 |
| runtime | 19 / 26 / 0 | 原基线及新 generation 对照通过；新增 1 缺 start 握手、18 非法 200、5 传输能力、2 重复派发红灯 |
| recovery | 10 / 0 / 0 | 既有 start 丢响应/generation/取消/并发基线 |
| core | 3 / 6 / 0 | 非法 contract_required 仍绕过 schema 类型校验 |
| views | 179 / 1 / 0 | 合同编辑入口仍缺失 |
| 测试设施（另计） | 10 / 0 / 0 | 8 附件指纹/摘要向量及 2 隔离 preflight，不算产品验收 |

核心证据：真实 runTask 对空 HTTP 200、缺失/未知/错误绑定 receipt 启动了假业务进程，marker=1、合成源码改变；原 HTTP 非成功拒绝仍不执行。合法 200 对照的执行探针有效，但实际请求未携带合同能力/准入对象，因此仍是握手红灯。未使用真实智能体、模型或生产材料。

实际 HTTP 单/批/legacy claim、注册和 WS 建连均未声明 task-contract-v1。真实 poller→handleTask→runTask 顺序重放同一 task/generation 执行假进程 2 次、complete 2 次；同批重复样本执行 1 次但多余 fail 1 次。不同 generation 的对照执行及 complete 各 2 次通过。这些是客户端实跑，不等于真实服务端允许重放。

真实附件下载接口阳性读取计数 1，跨工作区计数 0；新 OS 子进程重建真实 router、独立 PG pool，普通 Issue 重读通过。它们证明测试设施确实接线；合同内资源校验、完整合同重启持久性、事务故障、发布/claim/start 顺序仍停在缺入口/接点，不能用这些对照代替。

固定结果与命令：`evidence/wire-final/summary.json`；逐 TC/WS、叶子名称、SHA、分类、首败诊断：`evidence/wire-final/leaf-results.json`；原始日志：同目录各套件 stdout/stderr/JSON。完整 FR/A→FD→SP→TC 映射、夹具适配、局限、复现步骤在 `WIRE-SUPPLEMENT.md`。两份规格原文与已核验 SHA 保留在 `specs/`。

源码/非日志文档的 diff whitespace 检查、gofmt、JS 语法检查通过。根 ESLint 无配置的 exit 2 如实保留；没有声明仓库 lint 全绿。原始命令日志含终端空行，不改写原始输出去伪造 whitespace 全绿。首轮误用 testutil.Row.Scan 的编译错误保留且已修正，不计产品红灯；运行时夹具 changed_fields 自查修正后已重跑。传输追加首轮三项调度超时属于测试环境 helper 使用错误；原日志与诊断保留，切换现有真实进程内 execenv.Prepare 后在本 SHA 全套复跑。

性能（本 SHA，1 合成 Issue，1 预热 + 30 样本，毫秒）：

| 路径 | p50 | p95 | max |
| --- | ---: | ---: | ---: |
| 普通任务详情 | 1.179 | 1.578 | 1.655 |
| 普通任务列表 | 0.934 | 1.447 | 1.616 |

没有合同成功读取/准入/页面耗时，不作无退化或提速结论。首测和另一固定 SHA 的采样保留，未择优替换。

环境为 macOS arm64、Go 1.26.9、Node 23.9.0、pnpm 10.28.2。公共脚本建立专用 `multica_wush54_wire_supplement_839` 数据库并应用原有迁移至 594。测试结束后 `docker compose stop postgres` 已停止本轮启动的数据库容器，保留数据；未改变当前 daemon，没有后台测试或服务遗留。

浏览器只收集两项，未执行。显式 loopback 配置下 preflight exit 2，原因确认为缺少 TARS/Data 固定 SHA 隔离预览 manifest。仍欠真实 UI 完整表单/旧标签页/错误流程、服务端 WS 合同裁决、完整源码/文档元数据读取与默认上下文注入及合同 N 贯通；这些不能被本轮数量或部分绿灯替代。

任务保持 **in_progress**：本轮已按已核验 WS-01–WS-05 交付测试增量和证据，未声称完整测试阶段已验收，不再将产品缺入口/接点当需求级阻塞或整项 blocked。下一责任人为 Asurada 核验已交付红灯与局部接入顺序；实现、独立复核、集成和预览尚未启动。MOSS 没有实现产品 hook、事务、状态机或平行校验服务。

前次固定结果完整保留：`wire-fixed-04ed7271c`、`wire-fixed`，以及所有首败目录；最新只认 `wire-final`。源码 SHA 与最终证据提交之间只增加报告/日志，交付时再次核对。
