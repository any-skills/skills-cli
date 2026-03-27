![skills-cli 架构图](skills-cli-architecture.png)

# skills-cli

跨 AI Agent 的 Skills 统一管理工具，用 Go 实现，扩展本地 pull/push/watch 同步能力。

支持 40+ AI Agent（Cursor、Claude Code、Windsurf、Codex、Trae 等），统一管理 skill 文件。

## 安装

```bash
go install github.com/rushteam/skills-cli@latest
```

或从源码构建：

```bash
git clone https://github.com/rushteam/skills-cli.git
cd skills-cli
go build -o skills-cli .
```

## 快速开始

```bash
# 搜索远程 skills
skills-cli find typescript

# 从 GitHub 安装 skill
skills-cli add vercel-labs/agent-skills

# 查看已安装的 skills
skills-cli list

# 推送到所有 agent
skills-cli push --all
```

## 命令

### 远程生态

| 命令 | 说明 |
|------|------|
| `skills-cli find [query]` | 搜索 skills.sh 上的远程 skill |
| `skills-cli add <source>` | 从 GitHub/GitLab/本地路径安装 skill |
| `skills-cli check` | 检查已安装 skill 是否有更新 |
| `skills-cli update` | 更新所有已安装 skill |
| `skills-cli init [name]` | 创建 SKILL.md 模板 |
| `skills-cli remove [skills...]` | 移除已安装 skill |

### 本地同步

| 命令 | 说明 |
|------|------|
| `skills-cli list` | 列出所有已安装 skills（中央/全局/工程） |
| `skills-cli scan` | 扫描 agent 目录，显示同步状态（不拉取） |
| `skills-cli pull` | 从 agent 目录提取到中央目录 |
| `skills-cli push` | 从中央目录下发到 agent 目录 |
| `skills-cli watch` | 监听文件变化，自动同步 |

### 配置管理

| 命令 | 说明 |
|------|------|
| `skills-cli config show` | 显示当前配置 |
| `skills-cli config init` | 初始化默认配置 |
| `skills-cli config add-agent` | 添加自定义 agent |
| `skills-cli config remove-agent` | 移除 agent |
| `skills-cli config add-project <path>` | 注册工程目录 |
| `skills-cli config remove-project <path>` | 移除工程注册 |
| `skills-cli config list-projects` | 列出已注册工程 |
| `skills-cli config list-agents` | 列出所有 agent |
| `skills-cli config set-sync --agents a,b` | 设置默认同步 agent |
| `skills-cli config set-watch-direction <dir>` | 设置 watch 方向 |

## 源格式

`skills-cli add` 支持多种源格式：

```bash
# GitHub shorthand
skills-cli add vercel-labs/agent-skills

# GitHub URL
skills-cli add https://github.com/vercel-labs/agent-skills

# GitHub URL with subpath
skills-cli add https://github.com/vercel-labs/agent-skills/tree/main/skills/frontend-design

# GitLab URL
skills-cli add https://gitlab.com/org/repo

# 本地路径
skills-cli add ./my-local-skills

# 指定 skill
skills-cli add vercel-labs/agent-skills --skill frontend-design
```

## Pull / Push 同步

支持全局 agent 目录和工程级 agent 目录两个维度的同步。

```bash
# 从全局 cursor agent 拉取
skills-cli pull --agent cursor

# 从特定工程拉取（自动检测存在的 agent）
skills-cli pull --project /path/to/my-app

# 推送到所有注册工程 + 全局 agent
skills-cli push --all

# 仅查看 diff，不实际同步
skills-cli push --diff --agent cursor

# 强制覆盖，跳过冲突提示
skills-cli pull --force --all
```

## Watch 模式

```bash
# 默认：监听中央目录变化，自动推送到 agents
skills-cli watch

# 监听 agent 目录变化，自动拉取到中央
skills-cli watch --direction pull

# 双向监听
skills-cli watch --direction both
```

## 工程管理

```bash
# 注册工程
skills-cli config add-project /path/to/my-app

# 指定工程只同步特定 agent
skills-cli config add-project /path/to/my-app --agents cursor,claude-code

# 查看注册工程及检测到的 agent
skills-cli config list-projects
```

## 数据目录

| 路径 | 说明 |
|------|------|
| `~/.skills-cli/config.yaml` | 配置文件 |
| `~/.skills-cli/skills/` | 中央 skills 存储 |
| `~/.skills-cli/skill-lock.json` | 已安装 skill 的锁文件 |

## 支持的 Agent

Amp, Antigravity, Augment, Claude Code, OpenClaw, Cline, CodeBuddy, Codex, Command Code, Continue, Cortex, Crush, Cursor, Deep Agents, Droid, Firebender, Gemini CLI, GitHub Copilot, Goose, Junie, iFlow CLI, Kilo Code, Kimi CLI, Kiro CLI, Kode, MCPJam, Mistral Vibe, Mux, OpenCode, OpenHands, Pi, Qoder, Qwen Code, Replit, Roo Code, Trae, Trae CN, Warp, Windsurf, Zencoder, Neovate, Pochi, AdaL

## License

MIT
