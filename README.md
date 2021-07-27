<a href="https://terraform.io">
    <img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" alt="Terraform logo" title="Terraform" align="right" height="50" />
</a>

# Terraform Provider for Open Toolchains

Check `examples` folder for working examples

[documentation](https://registry.terraform.io/providers/dariusbakunas/opentoolchain/latest/docs)
## Development

### Requirements

- [Terraform](https://www.terraform.io/downloads.html) 0.13.6+ (to run acceptance tests)
- [Go](https://golang.org/doc/install) 1.15.8+ (to build the provider plugin)

### Quick Start

First, clone the repository:

```bash
git clone git@github.com:dariusbakunas/terraform-provider-opentoolchain.git
```

To compile the provider, run make build. This will build the provider and put the provider binary in the current directory.

```bash
make build
```

Run `make install` to install provider binary under `~/.terraform.d/plugins/ibm.com/dbakuna/opentoolchain/{VERSION}/{OS_ARCH}`.

After it is installed, terraform should be able to detect it during `terraform init` phase.

### Testing

To run unit tests, simply run:

```bash
make test
```

To run acceptance tests, make sure `IAM_API_KEY` environment variable is set and execute:

```bash
make testacc
```

**Note:** Acceptance tests create/destroy real resources, while they are named using `tf_acc_test` testing prefix, use some caution. Check `provider_test.go` contents for supported environment variables and their default values.

### Documentation

#### Environment setup

1. Install [pipenv](https://pipenv.readthedocs.io/en/latest/#install-pipenv-today)

```bash
% brew install pipenv
```

2. Install [pyenv](https://github.com/pyenv/pyenv#installation)

```bash
% brew install pyenv
```

3. Use `pyenv` to install `Python 3` if not installed already (run `pyenv versions` to check installed versions or `pyenv list` to list available for install):

```bash
% pyenv install 3.8.1
```

1. Run `pipenv install` to install `mkdocs` dependencies

2. In order to activate the virtual environment associated with this project you can simply use the shell keyword:

```bash
% pipenv shell
```

#### Generating docs

1. To generate or update Terraform documentation, run `go generate`.

2. To serve mk-docs locally, run `mkdocs serve`.

3. To push changes to `gh-pages` branch, run `mkdocs gh-deploy`


### Debugging

Run your debugger (eg. [delve](https://github.com/go-delve/delve)), and pass it the provider binary as the command to run, specifying whatever flags, environment variables, or other input is necessary to start the provider in debug mode:

```bash
make build-debug
dlv exec --listen=:54526 --headless ./terraform-provider-opentoolchain -- --debug
```

Note: IntelliJ may need additional flag `--api-version=2`

Connect your debugger (whether it's your IDE or the debugger client) to the debugger server. Example launch configuration for VSCode:

```json
{
    "apiVersion": 1,
    "name": "Debug",
    "type": "go",
    "request": "attach",
    "mode": "remote",
    "port": 54526, 
    "host": "127.0.0.1",
    "showLog": true,
    //"trace": "verbose",            
}
```

Have it continue execution and it will print output like the following to stdout:

```bash
Provider started, to attach Terraform set the TF_REATTACH_PROVIDERS env var:

        TF_REATTACH_PROVIDERS='{"dariusbakunas/opentoolchain":{"Protocol":"grpc","Pid":30101,"Test":true,"Addr":{"Network":"unix","String":"/var/folders/mq/00hw97gj08323ybqfm763plr0000gn/T/plugin900766792"}}}'
```

Copy the line starting with `TF_REATTACH_PROVIDERS` from your provider's output. Either export it, or prefix every Terraform command with it. Run Terraform as usual. Any breakpoints you have set will halt execution and show you the current variable values.
