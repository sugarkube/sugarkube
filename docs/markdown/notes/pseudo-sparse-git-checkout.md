# Sparse git checkouts
* If the target directory to install into doesn't exist, create it.
* If no git repo has been initialised there, init one.
* Set up the repo for sparse checkouts:
```
git remote add origin {{ git_url }}
git fetch
git config core.sparsecheckout true
echo '{{ source_path }}/*' > .git/info/sparse-checkout
git checkout {{ source_branch }}
git tag -v {{ source_branch }} 2>&1 >/dev/null | grep -E '{{ trusted_gpg_keys|join('|') }}'
```
* Don't run the last command if we don't care about verifying tags.
