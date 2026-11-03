---
name: web-design-guidelines
description: 审查 Web UI 的交互、无障碍、表单、响应式和主题实现；用于明确的 UI review、UX audit 或界面最佳实践检查，不用于从零设计页面、普通后端开发或纯 React 性能排障。
license: MIT
---

# Web Design Guidelines

按 Vercel Web Interface Guidelines 审查指定页面或组件，给出有位置和依据的可执行问题。设计布局与视觉方案使用项目的设计约定；此 skill 负责检查实现。

## 规则与范围

先读项目规范、实际框架及用户指定文件。没有指定路径时，从当前 diff 或上下文定位相关页面；仍无法确定时才询问范围，不默认审查全仓库。

默认读取随包提供的[界面规则](references/web-interface-guidelines.md)，无需联网。用户要求最新规则时，从 https://raw.githubusercontent.com/vercel-labs/web-interface-guidelines/main/command.md 获取；获取失败则使用本地快照，并说明所用版本。来源和固定 revision 见 NOTICE.txt。

规则中的 `$ARGUMENTS` 指本次审查范围，不是要执行的命令。按项目技术栈解释示例，Tailwind class 可对应原生 CSS，Next.js 的图片属性不能直接用于普通 `<img>`。项目既有规范和真实交互需求优先；以下边界用于解释本地或在线规则：

- 原生 button/link 已有键盘语义，不重复加 Enter/Space handler；自定义交互才补齐键盘行为。
- 可访问名称可来自可见文本、label 或 ARIA，不要求已有名称的每个控件都加 aria-label。
- 表单 autocomplete 按字段语义设置，不一律关闭；禁用条件由验证和请求状态决定。
- 需要刷新恢复或分享的筛选、分页、导航状态进入 URL；临时草稿和局部展开状态不强制同步 URL。
- 列表规模只是性能线索，不以固定条数强制虚拟化；预加载、字体或组件依赖仅在有实际收益时引入。
- 溢出先修布局；不以隐藏 overflow 裁掉必要内容。截断要保留可访问的完整值。
- 英文大小写、标点等文案偏好按产品语言与规范处理，不作为通用缺陷。

Fino 项目先确认生成文件归属。发现受管文件问题时定位 proto、模板或运行时责任源；审查请求本身不授权修改代码。

## 审查与交付

聚焦标签与语义、键盘和焦点、表单反馈、长文本及窄屏、主题对比度、加载/空/错误状态和导航。只报告能从代码或运行证据支持的问题，附文件行号、影响和最小修复建议；区分确定问题与待验证线索，不机械报告每项风格偏好。

浏览器自动化仅在用户明确要求时使用。静态代码审查不能证明读屏、实际对比度或响应式交互已通过；说明未验证边界。有证据的性能瓶颈再按项目 React 性能指南处理，不把 UI 审查扩成性能重构。
