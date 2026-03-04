<div align="center">
  <img src="assets/logo.png" alt="Operator OS" width="512">

  <h1>Operator OS</h1>

  <p><strong>The Ultra-Lightweight AI Agent Framework for Constrained Environments</strong></p>

  <p>
    <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/Architecture-x86__64%20%7C%20ARM64%20%7C%20RISC--V-blue" alt="Architecture">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
    <br>
    <a href="https://operator.onl"><img src="https://img.shields.io/badge/Website-operator.onl-blue?style=flat&logo=google-chrome&logoColor=white" alt="Website"></a>
    <a href="https://discord.gg/V4sAZ9XWpN"><img src="https://img.shields.io/badge/Discord-Community-4c60eb?style=flat&logo=discord&logoColor=white" alt="Discord"></a>
  </p>
</div>

---

**Operator OS** is an ultra-lightweight, high-performance personal AI Agent framework written in Go. Designed to run on hardware as inexpensive as $10 with a memory footprint of less than 10MB, Operator OS brings true continuous intelligence to the edge.

Built from the ground up to support autonomous agents, persistent memory, and multi-channel messaging, it bridges the gap between complex reasoning models and constrained runtime environments.

## ✨ Core Capabilities

- **Ultra-Lightweight Engine**: Consumes <10MB of RAM—99% smaller than typical Node.js or Python-based agent frameworks.
- **Lightning Fast Boot**: Cold starts in under 1 second, even on single-core 0.6GHz processors.
- **True Portability**: Deploys as a single, self-contained binary across RISC-V, ARM, and x86 architectures.
- **Continuous Memory**: Structural, persistent long-term memory that carries context seamlessly across sessions and reboots.
- **Multi-Channel Integration**: Natively supports Slack, Discord, Telegram, and WhatsApp out of the box.
- **Universal Model Support**: Drop-in support for OpenAI (5.x), Anthropic (Claude 4.x), Google Gemini (3.x Pro), and local Ollama.

## 🛠️ Typical Workflows

| Autonomous Engineering | System Automation | Information Retrieval |
| :---: | :---: | :---: |
| <img src="assets/operator_code.gif" width="240"> | <img src="assets/operator_memory.gif" width="240"> | <img src="assets/operator_search.gif" width="240"> |
| **Develop • Deploy • Scale**<br>Agents can access your local terminal and execute complex multi-step coding tasks. | **Schedule • Automate • Recall**<br>Native Cron tooling allows agents to run periodic health checks and background jobs. | **Discovery • Insights**<br>Agents can securely search the web and extract data without human intervention. |

## 🚀 Installation & Quick Start

### Precompiled Binaries
Download the appropriate binary for your system from our [Releases](https://github.com/avrilonline/Operator-OS/releases) page.

### Build from Source
Requires Go 1.21+.

```bash
git clone https://github.com/avrilonline/Operator-OS.git
cd Operator-OS
make deps
make build
```

### Initializing the Agent

**1. Initialize your workspace**
```bash
operator onboard
```

**2. Configure your API keys**
Edit `~/.operator/config.json` to link your preferred LLM and messaging channels:

```json
{
  "model_list": [
    {
      "model_name": "gpt-5.2",
      "model": "openai/gpt-5.2",
      "api_key": "sk-your-openai-key"
    },
    {
      "model_name": "claude-4-5-sonnet",
      "model": "anthropic/claude-4-5-sonnet-20260220",
      "api_key": "sk-ant-your-key"
    },
    {
      "model_name": "gemini-3.1-pro",
      "model": "gemini/gemini-3.1-pro",
      "api_key": "AIza-your-google-key"
    }
  ],
  "agents": {
    "defaults": {
      "model": "claude-4-5-sonnet"
    }
  },
  "channels": {
    "slack": {
      "enabled": true,
      "bot_token": "xoxb-YOUR_SLACK_BOT_TOKEN",
      "app_token": "xapp-YOUR_SLACK_APP_TOKEN"
    }
  }
}
```

**3. Start the Gateway**
Run the Gateway daemon to bring your agent online and connect it to your configured channels.
```bash
operator gateway
```

You can now message your agent directly via Slack (or your chosen channel) or interact with it via the CLI:
```bash
operator agent -m "What is the status of the system?"
```

## 🔌 Supported AI Providers

Operator supports a zero-code model configuration system. Simply prefix the model name with the provider protocol in your `model_list`.

| Provider | Protocol Prefix | Example |
|---|---|---|
| **Anthropic** | `anthropic/` | `anthropic/claude-4-5-sonnet-20260220` |
| **OpenAI** | `openai/` | `openai/gpt-5.2` |
| **Google Gemini** | `gemini/` | `gemini/gemini-3.1-pro` |
| **Groq** | `groq/` | `groq/llama3-8b-8192` |
| **Ollama (Local)** | `ollama/` | `ollama/llama3` |

## 🛡️ Security & Sandboxing

By default, Operator OS runs its agents in a secure sandbox.
- **Workspace Confinement**: Agents are restricted to reading/writing files within the configured workspace directory (default: `~/.operator/workspace`).
- **Command Filtering**: The `exec` tool blocks dangerous system commands (`rm -rf`, disk formatting, system shutdowns) even if workspace restrictions are bypassed.

To grant the agent full access to your host system (e.g., for system administration workflows), you must explicitly disable the sandbox in your configuration:

```json
{
  "agents": {
    "defaults": {
      "restrict_to_workspace": false
    }
  }
}
```
*Note: Only disable sandboxing in trusted, single-user environments.*

## 🐳 Docker Deployment

To run the Gateway completely containerized without installing Go locally:

```bash
git clone https://github.com/avrilonline/Operator-OS.git
cd Operator-OS

# Generate the default configuration structure
docker compose -f docker/docker-compose.yml --profile gateway up

# Edit the generated config file with your keys
vim docker/data/config.json

# Start the Gateway in the background
docker compose -f docker/docker-compose.yml --profile gateway up -d
```

## 🤝 Contributing

We welcome pull requests and issues! As Operator OS expands, we are looking for community maintainers to help build new skills, channel integrations, and deployment guides for edge hardware.

See the [Roadmap](docs/ROADMAP.md) for our current trajectory.

## 📄 License

This project is licensed under the [MIT License](LICENSE).