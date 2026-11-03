# Fino skills

这些技能随 `fino create` 安装，也可用 `fino create skills --output=<project-root>` 补装。`.agents/skills` 和 `.claude/skills` 指向本目录。默认保留已有文件；`--force` 会覆盖定制内容。

按任务选择技能，或在请求中明确指定名称；Codex 可使用 `$skill-name`。执行仍遵循项目规范和用户授权。

备选技能位于 `.optional/`，完整目录可移入本目录供项目使用。项目 `.gitignore` 中的 `/skills/.optional/` 只控制 Git 跟踪，不保证助手跳过技能发现；移回备选目录也不等于禁用技能。

| Skill | 用途 | 请求示例 | 来源 |
|---|---|---|---|
| [fino](fino/SKILL.md) | 契约、代码归属、实现、构建与发布 | `$fino 新增订单查询 RPC，并更新现有目标` | [Fino](https://github.com/fino-io/fino/tree/main/skills/generator/templates/skills/fino) |
| [frontend-design](frontend-design/SKILL.md) | 业务页面布局、交互和视觉设计 | `$frontend-design 优化订单列表和空状态` | [Anthropic](https://github.com/anthropics/skills/tree/main/skills/frontend-design)，业务界面适配 |
| [web-design-guidelines](web-design-guidelines/SKILL.md) | UI 实现与无障碍审查 | `$web-design-guidelines 审查订单页面的表单和键盘交互` | [Vercel skill](https://github.com/vercel-labs/agent-skills/tree/main/skills/web-design-guidelines) / [界面规则](https://github.com/vercel-labs/web-interface-guidelines)，附离线快照 |
| [vercel-react-best-practices](vercel-react-best-practices/SKILL.md) | 请求、包体积和 React 渲染性能 | `$vercel-react-best-practices 排查筛选输入卡顿` | [Vercel](https://github.com/vercel-labs/agent-skills/tree/main/skills/react-best-practices)，React/Vite 客户端适配 |
| [playwright](playwright/SKILL.md) | 浏览器操作、截图和 UI 流程验证 | `$playwright 验证筛选和详情返回流程` | [OpenAI](https://github.com/openai/skills/tree/main/skills/.curated/playwright)，含 [Microsoft](https://github.com/microsoft/playwright-cli/tree/main/skills/playwright-cli) 材料 |
| [gh-fix-ci](gh-fix-ci/SKILL.md) | GitHub Actions 失败诊断与修复 | `$gh-fix-ci 检查 PR 123 的失败原因` | [OpenAI](https://github.com/openai/skills/tree/main/skills/.curated/gh-fix-ci) |
| [security-best-practices](security-best-practices/SKILL.md) | Go、JS/TS 的安全审查与安全默认值 | `$security-best-practices 审查上传鉴权与输入校验` | [OpenAI](https://github.com/openai/skills/tree/main/skills/.curated/security-best-practices)，精简到项目常用栈 |
| [security-threat-model](security-threat-model/SKILL.md) | 资产、信任边界、攻击路径与控制措施 | `$security-threat-model 分析登录和文件上传流程` | [OpenAI](https://github.com/openai/skills/tree/main/skills/.curated/security-threat-model) |

`fino` 管工程与契约，设计、界面审查和性能技能按需组合。浏览器自动化仅在明确要求时执行；Playwright 需要 Node/npm/npx，CI 检查需要 Python 3 和已认证的 `gh`。安全技能用于安全任务，不作为普通开发的必经流程。

第三方版本、许可和本地适配见各目录的 `NOTICE.txt` 与 `LICENSE.txt`。上游链接的最新内容不等于已安装版本；构建通过也不代表视觉、无障碍或性能已实测。

## 备选技能

| Skill | 用途与调用示例 | 依赖 | 来源 |
|---|---|---|---|
| [ui-ux-pro-max](.optional/ui-ux-pro-max/SKILL.md) | 新产品的风格、配色、字体和设计系统；`$ui-ux-pro-max 为运营后台探索设计方向` | Python 3，标准库；已有主题优先复用 | [Next Level Builder](https://github.com/nextlevelbuilder/ui-ux-pro-max-skill)，MIT |
| [gh-address-comments](.optional/gh-address-comments/SKILL.md) | 读取并处理 PR 审查意见；`$gh-address-comments 修复当前 PR 的未解决意见` | Python 3、已认证的 `gh`、关联 PR | [OpenAI](https://github.com/openai/skills/tree/main/skills/.curated/gh-address-comments)，Apache-2.0 |
| [doc-coauthoring](.optional/doc-coauthoring/SKILL.md) | RFC、技术方案和架构决策；`$doc-coauthoring 编写订单服务拆分方案` | 本地 Markdown 即可，连接器可选 | [Anthropic](https://github.com/anthropics/skills/tree/main/skills/doc-coauthoring)，Apache-2.0 |
| [accessibility](.optional/accessibility/SKILL.md) | 深入无障碍审查；`$accessibility 审查弹窗焦点和表单错误提示` | 实测需浏览器；Lighthouse/axe 可选 | [web-quality-skills](https://github.com/addyosmani/web-quality-skills)，社区项目，MIT |
| [performance](.optional/performance/SKILL.md) | 加载、图片、字体、网络和浏览器性能；`$performance 排查首页加载缓慢` | 实测需浏览器和性能工具；DevTools MCP 可选 | [web-quality-skills](https://github.com/addyosmani/web-quality-skills)，社区项目，MIT |

移动时保留脚本、数据和引用，并按实际位置定位资源。重新安装会补齐备选目录中的缺失文件；默认保留定制内容，`--force` 会覆盖模板文件。

其他按需选项：

- [shadcn 官方技能](https://ui.shadcn.com/docs/skills)：项目采用 shadcn/ui 后安装。
- [Anthropic mcp-builder](https://github.com/anthropics/skills/tree/main/skills/mcp-builder)：开发 MCP 服务时使用，主要面向 TypeScript/Python。
- [OpenAI security-ownership-map](https://github.com/openai/skills/tree/main/skills/.curated/security-ownership-map)：团队安全责任分析，需要 Git 历史、Python/NetworkX。
- [Impeccable](https://github.com/pbakaus/impeccable)：使用官方 `npx impeccable install` 流程配置引擎和 hooks，完整能力不能仅靠拖拽技能目录启用。
- [TanStack Intent](https://tanstack.com/intent/latest)：目前为 alpha；采用 TanStack 后按实际 npm 依赖版本获取技能。
