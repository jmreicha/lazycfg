Goals

When working with a lot of different tools and environments, configuration sprawl quickly becomes a problem. How do you know what you need? How do you keep track of the chaos? Lazycfg aims to a be an all encompassing local configuration tool.

- [ ] Rewrite in Golang
- [ ] Use native Go libraries when possible instead of reaching out to the OS via process/commands
- [ ] Split the tool into multiple files, one per config to generate, to make it easier to extend
- [ ] Make the sso start url configurable but defaulted to Lytx url
- [ ] Add an option to have the tool update itself
- [ ] Add completions command for cli autocomplete
- [ ] Documentation site
- [ ] SSH configuration management
- [ ] Should have a yaml config file for complex declarative configs
- [ ] Set up ~/.granted/config with some sane defaults including `CredentialProcessAutoLogin = true` to automatically login when connecting. See https://docs.commonfate.io/granted/recipes/pass. If this is not enabled the following command must be run manually first `granted sso login --sso-start-url https://d-92670a73b3.awsapps.com/start --sso-region us-west-2`
- [ ] Replace EKS config generation tool with custom code
  - [ ] Generate AWS config and credentials file
- [ ] Option to modify kube config to use aws-vault for more secure auth
  - [ ] Option to merge existing kube config files instead so that there is no need to overwrite (kubectl config view --flatten)
    - [ ] https://github.com/kairen/kubectl-config-merge
    - [ ] https://github.com/QJoly/kubeconfig-merger
    - [ ] https://github.com/corneliusweig/konfig
    - [ ] https://github.com/dvob/kube-config-merge
- [ ] Use steampipe sdk to generate Kubernetes and AWS configs
  - [ ] Look at using the sdk to install all plugins when generating configs and make it easily configurable with these defaults
    - [ ] aws
    - [ ] cloudflare
    - [ ] crowdstrike
    - [ ] github
    - [ ] godaddy
    - [ ] jira
    - [ ] kubernetes
    - [ ] net (dns and certs)
    - [ ] newrelic
    - [ ] okta
    - [ ] pagerduty
    - [ ] steampipe
    - [ ] wiz
- [ ] Use commonfate/granted sdk to generate configs
- [ ] Better handling of kube configs if non-standard configs exist by parsing the file first
- [ ] Comprehensive tests and test coverage
- [ ] Universal installer and packages for different platforms
  - [ ] Brew
  - [ ] Docker
  - [ ] Deb
  - [ ] Pacman
  - [ ] Windows

Steampipe configs

https://github.com/turbot/pipes-sdk-go
https://github.com/turbot/steampipe-export

Granted configs

https://github.com/common-fate/sdk

Future ideas and improvements

- Brew support
- Automatically generate gitignore files https://github.com/jasonuc/gignr
- Docs
  - Why create this tool? Introduction. Example here https://docs.commonfate.io/granted/introduction
  - Installation
  - Usage
