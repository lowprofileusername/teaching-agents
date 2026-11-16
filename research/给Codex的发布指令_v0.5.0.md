# 给 Codex 的发布指令：自查 → 深度融入 → v0.5.0 发布（Actions）

> 用法：整段粘贴给 Codex。保留两个确认门（自查表、PR 合并），其余已授权。
> 背景：独立核验（fix/independent-audit）与撤回修正、revise 模式、字级 Word 写回已在 feat/ai-track-builder 完成。本指令是收尾与发布，不是从零改造。

---

你在仓库 `lowprofileusername/teaching-agents` 工作。目标：完成执行前自查与剩余接线，然后用 GitHub Actions 发布 v0.5.0。按阶段执行，**不要跳过确认门**。

## 阶段 0 · 前提
1. `git fetch`。确认 `fix/independent-audit` 已合并进 main（**未合并就停下来告诉我**）。检出 `feat/ai-track-builder`，把 main 合并进来（带入 `knowledge/ai_track_deep.json`、`research/source-evidence/`、修正后的 research MD）。冲突处理原则：main 的修正版数据/文档 + 本分支的 builder/revise/字级写回代码，两边功能都保留，不得删改任一侧。
2. 读 `research/给Codex的当前状态与剩余任务_2026-11-17.md` 与 `research/独立核验报告_2026-11-17.md`。
3. **撤回指令的 5 项修正已实现（commit f264905），不要重做**；只验证 `TestBuilderAuditRetractionRegressions` 通过。revise 修订+批注模式（3f071b8）与字级 Word 写回（a7ffb3b）**不得回退**。

## 阶段 1 · 执行前自查（先发表格，等我确认）
按「PASS / FAIL + 证据位置」列表发我：
1. 抽查 `priority_sessions`（李静/宋晓菲）与 `support_sessions`（于歆杰/汪琼/刘学莘）每个 move 的 `source_locator` 能否在 `research/source-evidence/` 对应文件找到原文。独立核验报告已全量核验过，你抽查关键 move 并与报告对照；**发现任何与报告不符＝FAIL**。
2. `sixth_edition_reports` 7 条：`source_url` 与 `verification` 合理；transcribed 不得当逐字原话；高越「三维痛点定位法/AI适配度象限」不得编造具体维度。
3. `ai_track_deep.json` 无编造成效数字（人数/提升率/分数/获奖归因到用户课程）。
4. 5 项撤回已在代码落实：伦理治理有落点即可（不强制独立章）、课堂实录 40–50 分钟、说课课前课后必须（课中衔接可选）、四项标注只覆盖有 AI 动作环节、案例逐条 source_url/verification。
5. `go test ./...`、`node tests/builder.mjs`、`node tests/revise.mjs`、`python verify_codex_work.py .` 全绿。

**任何 FAIL：停下来报告，等我确认；不要继续，也不要自己「合理补写」。全部 PASS 且我确认后，才进阶段 2。**

## 阶段 2 · 剩余接线（本次发布范围，三项全做）
**任务 1（最高优先）**：执行 `research/给Codex的AI赛道深度融入任务书.md`（修正版：2+3 场共 19 个 move，不是旧写的「10 个打法」；案例示例是石拓不是张琴）：
- embed `knowledge/ai_track_deep.json`；`builtin:aideep:*` 只读检索条目（2 priority + 3 support + 7 reports）。
- architect/video-planner 生成优先引用李静/宋晓菲 move；support 按需；第六届案例三重标注（source_nature / 往届公开报道非你的课程事实 / transcribed 不得当原话）；高越禁编维度；与 ai_playbook 去重。
- ai 顾问追加对照检查；测试按任务书任务 4（含反向用例）。

**任务 2**：执行 `research/给Codex的下一步增强指导_报告与PPT.md`：
- architect `depth:"full"` 逐段扩写：`BuilderSection` 加可选 `draft`（不新增顶层字段，保 DisallowUnknownFields）；成效数字/学生原话/获奖一律"待补："占位，`generatedFactRisk` 硬校验；字数按节预算、合计≤4000；伦理内容按落点口径（不强制独立章）。
- 新增 `presentation-architect`（注册 teachingSpecialists + isBuilder）：`BuilderPlan.slides` 可选数组（id/page_no/title/section/bullets≤4/visual/evidence_source/speaker_note/duration_sec）。**硬校验只有官方 ≤12 分钟（≤720 秒）**+ slides 非空 + 必填字段 + evidence_source 在本轮 available；**≤30 页（众师云 12923 第三方建议）、3+6+3（AI 综合无原文出处）、690–710 秒目标一律只出 warning，不得拒绝**；匿名（不得出现姓名/学校/院系）；测试 TestPresentationArchitectPlan（721 秒拒、虚构 evidence_source 拒、30 页/3+6+3/690–710 只 warning）。

**任务 3**：执行 `research/给Codex的下一步增强指导_赛道专属方法论.md`（修正版）：
- 新建 `knowledge/track_playbooks.json` + embed + 按赛道检索；generationPrompt 按 `p.Track` 注入 mindset/plays，architect 引用 1–2 个打法并标"迁移建议非专家本人观点"；video-planner 行为不变。
- **赛道归属以 00总账§三为准**：新农科＝向妙莲（不得留空）、实验教学＝刘战合、课程思政含叶齐、产教融合＝俞梅芳、新工科＝刘学莘；熊芸/卢安总账未记载参赛赛道，按学科归类须标"分析者判断"。caution 照抄 02§六对应行原文；确无出处的赛道留空标 `no_traceable_source`，生成时明说"暂无可引用打法，仅通用建议"，不得跨赛道改名顶替或虚构讲者。
- 测试按指导任务 4（新农科取到向妙莲；新医科辛页/姜宏佺；新工科刘学莘/熊芸；实验教学刘战合）。

数据纪律不变：不虚构用户成效；引用带来源与性质标注；字数只提醒不自动删减。三项任务的测试与原 10 PASS、TestBuilderTrackSwitchAndTwoVideoRequirements、撤回回归、revise 测试必须全绿。

## 阶段 2.5 · 桌面端 UI 验收（你执行，发布门禁）
照 `tests/builder-ui.cjs` 的"隔离服务器 + 本地 mock 模型 + Playwright"模式（启动方式见 `docs/AI_TRACK_BUILDER.md` 端到端节），新建 `tests/revise-ui.cjs`：
1. `go build -o "$QA/test-server" .`；用 `TEACHING_AGENT_PROFILES` / `TEACHING_LIBRARY_DB` 指向隔离目录启动（`-no-browser`），从日志读带 token 的 URL；mock 模型见到"修订+批注"指令时返回确定性 revise JSON（revisions 区间与夹具 docx 原文逐字一致）。
2. 夹具 docx：一个普通段落（含多条修订+批注）、一个含 drawing 的复杂段落、材料总字数 >4000（触发字数提醒）。
3. UI 断言：上传材料 → 点"修订+批注（可接受/拒绝）"→ `#reviseDialog` 显示修订/批注/字数提醒（**仅提醒、不阻断**）；接受部分、拒绝部分 → `#reviseStatus` 计数正确 → "写回 Word" → 导出状态含"字级修订 N 处"。
4. 解析导出 docx 断言：已接受区间出现字级 `w:del/w:ins`（文本一致）、被拒修订未写回、批注含批注+修订明细、复杂段落正文不变且有降级 warning；截图面板与导出对话框，写 `ui-results.json`。
5. 同模式覆盖新功能：presentation-architect 用 mock 生成 PPT 方案 → 渲染断言 + 721 秒被拒 + 30页/3+6+3/690–710 只 warning 不拒绝；赛道打法检索命中（新工科刘学莘、新农科向妙莲、实验教学刘战合）。
6. 把 PASS/FAIL 表 + 截图 + 导出样本文件发我。**任何 FAIL 停下来报告。本阶段与阶段 3 回归全绿才允许打 tag。**
边界说明：验证的是 UI 交互与修订的 OOXML 正确性；Word 里红线视觉渲染由用户打开导出样本抽查，你负责交付样本。

## 阶段 3 · 发布 v0.5.0（GitHub Actions）
1. `frontend/package.json` version 改 `0.5.0`；`RELEASE_NOTES.md` 顶部新增 v0.5.0 章节：AI 赛道 builder（architect/video-planner，赛道感知）、revise 修订+批注（字级修订可接受/拒绝、字数只提醒）、字级 Word 写回（复杂段落降级批注）、AI 赛道深度融入（李静/宋晓菲优先）、报告逐段扩写（depth:full，成效留"待补"）、presentation-architect PPT 生成（硬校验仅官方≤12分钟，30页/3+6+3/690–710秒为建议只warning）、赛道专属打法库（track_playbooks.json，出处可回查）、独立审计修正（78 处，见核验报告）。
2. `.github/workflows/release.yml` 的 node 测试列表补 `node tests/builder.mjs` 与 `node tests/revise.mjs`（与 builder-checks.yml 对齐）。
3. 全量回归：`go test ./...`；release.yml 列出的全部 node 测试 + builder + revise；`python verify_codex_work.py .`；**阶段 2.5 的 revise-ui 验收 PASS**。**开 PR（不要自己合并 main）**，附自查表、UI 验收表与测试结果。
4. **我合并 PR 后**：在 main 的合并提交上打 tag `v0.5.0` 并推送 tag，触发 release 工作流（Windows：测试+构建双 persona exe+smoke/透明度→draft release；macOS：构建+smoke→上传 zip 并转正式）。
5. 等两个 job 都 success；核对 release 资产齐全（2 个 exe + macOS zip + SHA256SUMS + RELEASE_NOTES），把 release 链接与资产清单发我。**任何一步失败，报告日志，不要绕过流水线手动发布。**