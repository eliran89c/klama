<div align="center">
  <img src="images/logo.svg" alt="Klama Logo" width="200">
</div>

# Klama - AI-powered DevOps Debugging Assistant

Klama is a CLI tool that helps diagnose and troubleshoot DevOps-related issues using AI-powered assistance. It interacts with language models to interpret user queries, suggest and execute safe commands, and provide insights based on the results. Currently, Klama supports Kubernetes (K8s) debugging, with plans to expand to other DevOps domains in the future.

## How it works

1. Klama sends your DevOps-related query to the AI model.
2. The AI, acting as a DevOps expert, interprets the query and may suggest commands to gather more information.
3. If a command is suggested, Klama will ask for your approval before executing it.
4. The command is executed if approved, and the output is sent back to the AI for further analysis.
5. This process repeats until the AI has enough information to provide a final answer.
6. Klama presents the AI's findings and any relevant information.

This approach ensures safety and gives users full control over the commands run in their environment.

## Requirements

- Go 1.22 or higher
- Access to a Kubernetes cluster (for K8s-related command execution)

## Installation

You can install Klama directly from GitHub:

```
go install github.com/eliran89c/klama@latest
```

This will download the source code, compile it, and install the `klama` binary in your `$GOPATH/bin` directory. Make sure your `$GOPATH/bin` is in your system's PATH.

## Configuration

Klama requires a YAML configuration file to set up the AI model. The configuration file is searched for in the following order:

1. Custom location specified by the `--config` flag
2. `$HOME/.klama.yaml`
3. `.klama.yaml` in the current directory

A valid configuration file with at least the required fields must be present for Klama to function properly.

### Required Configuration

The following fields are required in your configuration:

- `agent.name`: The name of the AI model
- `agent.base_url`: The API endpoint for the AI model

Klama will not run if these required fields are missing from the configuration file.

### OpenAI API Compatibility

Klama requires an OpenAI or OpenAI-compatible server to function. The application has been tested with the following frameworks and services:

- OpenAI models
- Self-hosted models using [vLLM](https://medium.com/@eliran89c/how-to-deploy-a-self-hosted-llm-on-eks-and-why-you-should-e9184e366e0a)
- Amazon Bedrock models via [Bedrock Access Gateway](https://github.com/aws-samples/bedrock-access-gateway)

While these have been specifically tested, any server that implements the OpenAI API should be compatible with Klama.

### Sample Configuration File (.klama.yaml)

Create a file named `.klama.yaml` in your home directory or in the directory where you run Klama. Here's an example of what the file should contain:

```yaml
agent:
  name: "anthropic.claude-3-5-sonnet-20240620-v1:0"  # Required
  base_url: "https://bedrock-gateway.example.com/api/v1"  # Required
  auth_token: ""  # Set via KLAMA_AGENT_TOKEN environment variable
  pricing: # Optional, will be used to calculate session price
    input: 0.003  # Price per 1K input tokens (optional)
    output: 0.015 # Price per 1K output tokens (optional)
```

### Environment Variables

You can set the authentication token using an environment variable:

- `KLAMA_AGENT_TOKEN`: Set the authentication token for the agent model

Example:
```sh
export KLAMA_AGENT_TOKEN="your-agent-token-here"
```

### Command-Line Configuration

You can specify a custom configuration file location using the `--config` flag:

```sh
klama --config /path/to/your/config.yaml k8s
```

## Usage

Currently, Klama provides one main subcommand:

### `k8s`: Interact with the Kubernetes debugging assistant

Run Klama with the `k8s` subcommand to start a Kubernetes debugging session:

```sh
klama k8s
```

This will start an interactive session where you can ask Kubernetes-related questions and get AI-powered assistance.

### Flags

- `--config`: Specify a custom configuration file location
- `--debug`: Enable debug mode

Example with flags:
```sh
klama k8s --debug --config /path/to/config.yaml
```

If Klama fails to start due to missing or invalid configuration, it will provide an error message indicating the issue. Ensure that your configuration file is properly formatted and contains all required fields before running Klama.

## Future Developments

While Klama currently focuses on Kubernetes debugging, there are plans to expand its capabilities to cover other DevOps domains in the future. Stay tuned for updates!

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.