
<a name="v0.13.0"></a>
## [v0.13.0](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.12.0...v0.13.0)

> 2021-09-21

### Feat

* add pagerduty datasource
* add initial pagerduty integration resource

### Pull Requests

* Merge pull request [#5](https://github.com/dariusbakunas/terraform-provider-opentoolchain/issues/5) from dariusbakunas/more-integrations
* Merge pull request [#4](https://github.com/dariusbakunas/terraform-provider-opentoolchain/issues/4) from dariusbakunas/tekton-pipeline-resource


<a name="v0.12.0"></a>
## [v0.12.0](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.11.2...v0.12.0)

> 2021-09-17

### Feat

* add import support for tekton pipeline
* add import support for slack integration
* add import support for keyprotect
* add github integration import support
* add tekton pipeline datasource
* add slack integration datasource
* add keyprotect integration data source
* add github integration datasource
* add slack integration resource
* add secret_env property to the tekton pipeline resource
* add text_env property to tekton pipline resource
* add keyprotect integration
* tekton pipeline triggers resource
* add tekton pipeline definition resource
* add pipeline definition schema
* enable github integration resource
* initial github integration resource
* initial tekton pipeline resource

### Fix

* update opentoolchain-sdk version


<a name="v0.11.2"></a>
## [v0.11.2](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.11.1...v0.11.2)

> 2021-08-13


<a name="v0.11.1"></a>
## [v0.11.1](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.11.0...v0.11.1)

> 2021-08-12


<a name="v0.11.0"></a>
## [v0.11.0](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.10.1...v0.11.0)

> 2021-08-12

### Feat

* add ability to override trigger branch/pattern

### Pull Requests

* Merge pull request [#3](https://github.com/dariusbakunas/terraform-provider-opentoolchain/issues/3) from dariusbakunas/trigger-branch


<a name="v0.10.1"></a>
## [v0.10.1](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.10.0...v0.10.1)

> 2021-08-06


<a name="v0.10.0"></a>
## [v0.10.0](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.9.1...v0.10.0)

> 2021-08-06

### Feat

* deprecate redundant data sources
* add combined datasource for getting tekton pipeline information


<a name="v0.9.1"></a>
## [v0.9.1](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.9.0...v0.9.1)

> 2021-08-06

### Feat

* add deprecation warnings for separate property/trigger resources


<a name="v0.9.0"></a>
## [v0.9.0](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.8.4...v0.9.0)

> 2021-08-05

### Feat

* add new resource that would combine properties and triggers

### Fix

* fix unit test
* clear new properties that are removed on update


<a name="v0.8.4"></a>
## [v0.8.4](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.8.3...v0.8.4)

> 2021-08-05

### Fix

* cleanup any new properties introduced


<a name="v0.8.3"></a>
## [v0.8.3](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.8.2...v0.8.3)

> 2021-08-05

### Fix

* hookId has inconsistent types, disable it for now


<a name="v0.8.2"></a>
## [v0.8.2](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.8.1...v0.8.2)

> 2021-08-05

### Fix

* regression for delete method, it was deleting all properties


<a name="v0.8.1"></a>
## [v0.8.1](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.8.0...v0.8.1)

> 2021-08-04


<a name="v0.8.0"></a>
## [v0.8.0](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.7.2...v0.8.0)

> 2021-08-04

### Feat

* add pipeline triggers resource
* add datasource for pipeline triggers

### Fix

* switch to newest client

### Pull Requests

* Merge pull request [#2](https://github.com/dariusbakunas/terraform-provider-opentoolchain/issues/2) from dariusbakunas/triggers


<a name="v0.7.2"></a>
## [v0.7.2](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.7.1...v0.7.2)

> 2021-07-30

### Feat

* update original properties whenever new input keys are added

### Fix

* make sure to cleanup original props that are no longer overriden
* only save original properties that have matching inputs


<a name="v0.7.1"></a>
## [v0.7.1](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.7.0...v0.7.1)

> 2021-07-30


<a name="v0.7.0"></a>
## [v0.7.0](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.6.11...v0.7.0)

> 2021-07-30

### Feat

* add deleted_keys and original_properties


<a name="v0.6.11"></a>
## [v0.6.11](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.6.10...v0.6.11)

> 2021-07-28

### Fix

* better handling for pipeline secrets


<a name="v0.6.10"></a>
## [v0.6.10](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.6.9...v0.6.10)

> 2021-07-28

### Reverts

* Formatting changes and removal of unused imports.

### Pull Requests

* Merge pull request [#1](https://github.com/dariusbakunas/terraform-provider-opentoolchain/issues/1) from bfelaco/master


<a name="v0.6.9"></a>
## [v0.6.9](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.6.8...v0.6.9)

> 2021-07-27

### Fix

* secret update


<a name="v0.6.8"></a>
## [v0.6.8](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.6.7...v0.6.8)

> 2021-07-27

### Fix

* make sure secrets are applied when there are no textEnv values


<a name="v0.6.7"></a>
## [v0.6.7](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.6.6...v0.6.7)

> 2021-07-26

### Feat

* add secret_env support


<a name="v0.6.6"></a>
## [v0.6.6](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.6.5...v0.6.6)

> 2021-07-15

### Fix

* do not call patch api if textEnv is not specified


<a name="v0.6.5"></a>
## [v0.6.5](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.6.4...v0.6.5)

> 2021-07-13


<a name="v0.6.4"></a>
## [v0.6.4](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.6.3...v0.6.4)

> 2021-07-13


<a name="v0.6.3"></a>
## [v0.6.3](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.6.2...v0.6.3)

> 2021-07-13

### Feat

* add optional repository_token parameter

### Fix

* make sure code remains backwards compatible


<a name="v0.6.2"></a>
## [v0.6.2](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.6.1...v0.6.2)

> 2021-07-09

### Fix

* temporary remove secret_env


<a name="v0.6.1"></a>
## [v0.6.1](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.6.0...v0.6.1)

> 2021-07-08

### Fix

* fix env property handling


<a name="v0.6.0"></a>
## [v0.6.0](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.5.0...v0.6.0)

> 2021-07-08

### Feat

* add computed services property to toolchain resource
* initial tekton pipeline datasource


<a name="v0.5.0"></a>
## [v0.5.0](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.4.0...v0.5.0)

> 2021-06-09

### Feat

* add support for tags
* add debugging support


<a name="v0.4.0"></a>
## [v0.4.0](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.3.1...v0.4.0)

> 2021-06-02

### Feat

* add generated url property


<a name="v0.3.1"></a>
## [v0.3.1](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.3.0...v0.3.1)

> 2021-05-27

### Reverts

* chore: lock goreleaser version


<a name="v0.3.0"></a>
## [v0.3.0](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.2.1...v0.3.0)

> 2021-05-26

### Feat

* add ability to update toolchain name
* additional env variable support to match IBM provider

### Fix

* switch to correct opentoolchain-go-sdk version


<a name="v0.2.1"></a>
## [v0.2.1](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.2.0...v0.2.1)

> 2021-05-25

### Feat

* include error details


<a name="v0.2.0"></a>
## [v0.2.0](https://github.com/dariusbakunas/terraform-provider-opentoolchain/compare/v0.1.0...v0.2.0)

> 2021-05-25

### Feat

* initial toolchain resource
* remove service/tpl props for now from datasource


<a name="v0.1.0"></a>
## v0.1.0

> 2021-05-24

### Feat

* additional toolchain datasource properties
* add initial toolchain datasource

