# tpf

`tpf` is a small utility to filter out specific data from `terraform plan` output.

This tool is intended to be used within [Command Output](https://github.com/marketplace/actions/command-output) github action so its output could be later fed to [Add PR Comment](https://github.com/marketplace/actions/add-pr-comment) github action without hitting the comment size limitation.

## Usage

```
usage: terraform show terraform.tfplan | tpf [-f file] [-e] [-d]
  -d, --diff=false: convert to diff globally
  -e, --eot=false: hide EOT blocks globally
  -f, --file="tpf.yaml": file with filter rules
```

The tool relies on the `.yaml` config with the filter rules to apply.

```yaml
# resource provider e.g. `helm_release`
resource1:
  # resource name e.g. `argocd`
  resource1_name1: 'regex for objects' # object regex e.g. `customresourcedefinition\.apiextensions\.k8s\.io.+`
resource2:
  resource2_name1: 'regex for objects'
  resource2_name2: 'regex for objects'
```

## Example

This is how we can filter out huge k8s CRD blocks for `helm_release` resource.

```bash
❯ cat filter.yaml
helm_release:
  argocd: 'customresourcedefinition\.apiextensions\.k8s\.io.+'

❯ terraform show terraform.tfplan | tpf -f filter.yaml
```

The matched objects get folded while the rest of the diff is not affected.

```hcl
~ resource "helm_release" "argocd" {
    #
    # reducted for example
    #
    ~ "customresourcedefinition.apiextensions.k8s.io/apiextensions.k8s.io/v1/applications.argoproj.io"             = {
      # (930 lines hidden: 0 to add, 1 to change, 0 to destroy)
    }
    ~ "customresourcedefinition.apiextensions.k8s.io/apiextensions.k8s.io/v1/applicationsets.argoproj.io"          = {
      # (3703 lines hidden: 0 to add, 1 to change, 0 to destroy)
    }
    ~ "customresourcedefinition.apiextensions.k8s.io/apiextensions.k8s.io/v1/appprojects.argoproj.io"              = {
      # (41 lines hidden: 0 to add, 1 to change, 0 to destroy)
    }
    #
    # reducted for example
    #
  }
```

## Inspirations

* https://github.com/dflook/terraform-github-actions/blob/main/image/src/github_pr_comment/plan_formatting.py
* https://gist.github.com/Kenterfie/a7ec9e50f17a749b8bb6469f21a6be4f
