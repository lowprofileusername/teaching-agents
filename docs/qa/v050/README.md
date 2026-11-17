# v0.5.0 发布验收

基线：main c282b75。阶段0/1自查见 [preflight](../v050-preflight.md)。

| 验收项 | 结果 | 证据 |
|---|---|---|
| Go全量（撤回、revise、字级写回、三项接线） | PASS | [日志](final-go.log) |
| 发布工作流14个Node套件 | PASS | [结果](final-node-results.json) |
| verify_codex_work.py | 13 PASS / 0 FAIL / 0 RED FLAG | [日志](final-verify.log) |
| 修订面板：多条修订、接受/拒绝、超4000字只提醒 | PASS | [截图](revise-panel.png) |
| Word字级修订、被拒修订不写回、复杂段落降级批注 | PASS | [导出截图](word-export.png)、[DOCX样本](word-review-sample.docx) |
| PPT方案渲染、721秒拒绝、31页/715秒/非3+6+3仅提醒 | PASS | [方案](presentation-plan.png)、[提醒](presentation-warnings.png) |
| architect depth:full正文、实际API请求 | PASS | [UI结果](ui-results.json) |
| 新工科/新农科/实验教学实际API检索 | PASS | [UI结果](ui-results.json) |
| 旧builder、新文科切换、双视频、无API、历史恢复/新建重置 | PASS | [旧功能UI回归](builder-ui-results.json) |

## 验收范围

隔离服务器、各自独立配置/历史目录、本地确定性mock模型、Chromium实际点击操作。未连接用户API或改写讲座库。验收UI和导出OOXML正确性；Word红线视觉渲染由使用者打开样本抽查，非真实模型输出质量认证。

早期失败已修复：短方法名检索门控、深度资料挤占其他打法、测试脚本误假定批注部件文件名、测试实例共享历史目录。原始失败日志保存在SMB QA目录，不改写成PASS。修订ID新增段落隔离测试，防止模型复用r1导致跨段落选择串联。

## 接线清单

- `knowledge/ai_track_deep.json`：embed并实际读取；12个 `builtin:aideep:*` 只读条目，保留原数据、source_locator、source_nature、verification和逐条报道链接。AI生成优先李静/宋晓菲，非AI生成排除，重叠打法去重。
- `knowledge/track_playbooks.json`：8个非AI赛道17条打法，`builtin:track:*`按赛道检索，AI沿用原库。机制和caution逐行核对02§六；分析者归类和跨赛道迁移单列。
- `architect`：可选depth=full，每节draft受预算和事实风险校验；不新增草稿顶层字段。
- `presentation-architect`：新增角色、可选slides与warnings；每页实际来源、匿名、字段/要点检查，总时长硬上限720秒，建议性限制不作拒绝条件。
- 原8顾问、7工作区及三个审阅分支保留；video-planner沿用撤回后的40–50分钟、课前/课后说课与AI环节标注要求。

GitHub Actions双平台构建/运行结果和发布资产以最终release为准；不绕过流水线手工发布。
