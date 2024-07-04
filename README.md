<div align="center">
  <img src="images/logo.svg" alt="Klama Logo" width="200">
</div>

# Klama - Kubernetes Debugging Assistant

Klama is a CLI tool that helps diagnose and troubleshoot Kubernetes-related issues using AI-powered assistance. It interacts with language models to interpret user queries, validate and execute safe Kubernetes commands, and provide insights based on the results.

## How it works

1. Klama sends your query to the main AI model.
2. The AI interprets the query and may suggest Kubernetes commands to gather more information.
3. If a command is suggested, Klama will validate it for safety using either:
   - A separate AI model (if provided in the configuration)
   - User approval (if no validation model is configured)
4. The command is executed if deemed safe, and the output is sent back to the main AI for further analysis.
5. This process repeats until the AI has enough information to provide a final answer.
6. Klama presents the AI's findings and any relevant Kubernetes information.

This approach allows for flexibility in model selection. A more capable model can be used for the main logic, while a faster, lighter model can optionally be used for command validation, potentially saving costs and increasing speed. If no validation model is provided, Klama will ask the user to approve each command before execution, ensuring safety and giving users full control over the commands run in their Kubernetes environment.

## Requirements

- Go 1.22 or higher
- Access to a Kubernetes cluster (for actual command execution)

## Installation

You can install Klama directly from GitHub:

```
go install github.com/eliran89c/klama@latest
```

This will download the source code, compile it, and install the `klama` binary in your `$GOPATH/bin` directory. Make sure your `$GOPATH/bin` is in your system's PATH.

## Configuration

Klama requires a YAML configuration file to set up the AI models. The configuration file is searched for in the following order:

1. Custom location specified by the `--config` flag
2. `$HOME/.klama.yaml`
3. `.klama.yaml` in the current directory

A valid configuration file with at least the required fields must be present for Klama to function properly.

### Required Configuration

The following fields are required in your configuration:

- `agent.name`: The name of the main AI model
- `agent.base_url`: The API endpoint for the main AI model

Klama will not run if these required fields are missing from the configuration file.

### OpenAI API Compatibility

Klama requires an OpenAI or OpenAI-compatible server to function. The application has been tested with the following frameworks and services:

- OpenAI models
- Self-hosted models using [vLLM](https://github.com/vllm-project/vllm)
- Amazon Bedrock models via [Bedrock Access Gateway](https://github.com/aws-samples/bedrock-access-gateway)

While these have been specifically tested, any server that implements the OpenAI API should be compatible with Klama.

### Sample Configuration File (.klama.yaml)

Create a file named `.klama.yaml` in your home directory or in the directory where you run Klama. Here's an example of what the file should contain:

```yaml
agent:
  model:
    name: "anthropic.claude-3-5-sonnet-20240620-v1:0"  # Required
    base_url: "https://bedrock-gateway.example.com/api/v1"  # Required
    auth_token: ""  # Set via KLAMA_AGENT_TOKEN environment variable
    pricing: # Optional, will be used to calculate session price
      input: 0.003  # Price per 1K input tokens (optional)
      output: 0.015 # Price per 1K output tokens (optional)

validation: # Comment this block out to manually approve the agent commands
  model:
    name: "meta-llama/Meta-Llama-3-8B"
    base_url: "https://vllm.example.com/v1"
    auth_token: ""  # Set via KLAMA_VALIDATION_TOKEN environment variable
    # pricing:
    #   input: 0
    #   output: 0
```

If the validation model is not specified, Klama will prompt the user to approve each command before execution.

### Environment Variables

You can set the authentication tokens using environment variables:

- `KLAMA_AGENT_TOKEN`: Set the authentication token for the agent model
- `KLAMA_VALIDATION_TOKEN`: Set the authentication token for the validation model

Example:
```sh
export KLAMA_AGENT_TOKEN="your-agent-token-here"
export KLAMA_VALIDATION_TOKEN="your-validation-model-token-here"
```

### Command-Line Configuration

You can specify a custom configuration file location using the `--config` flag:

```sh
klama --config /path/to/your/config.yaml "Your Kubernetes query here"
```

## Usage

Run Klama with your Kubernetes-related query:

```sh
klama [flags] <prompt>
```

For example:
```sh
klama "Why is my pod not starting?"
```

### Flags

- `--config`: Specify a custom configuration file location
- `--debug`: Enable debug mode
- `--show-usage`: Show usage information

Example with flags:
```sh
klama --debug --config /path/to/config.yaml "Check the status of all pods"
```

If Klama fails to start due to missing or invalid configuration, it will provide an error message indicating the issue. Ensure that your configuration file is properly formatted and contains all required fields before running Klama.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.