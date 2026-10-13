Release Flow:

1. Merge the PR into main in the web UI.
2. Tag the merge commit and push the tag:
```shell
git checkout main
git pull
git tag v0.1.0
git push origin v0.1.0
```
3. The tag push triggers the workflow: test, build all six targets, then create
   the release with the archives and checksums attached.

