#!/bin/bash
echo "$1"
git tag "$1" && git push --tags
gox -output="release/{{.Dir}}_{{.OS}}_{{.Arch}}" -osarch="darwin/386 darwin/amd64 linux/386 linux/amd64 linux/arm"
github-release release --user ad --repo gomgr --tag "$1" --name "$1"
github-release upload --user ad --repo gomgr --tag "$1" --name "gomgr_darwin_386" --file release/gomgr_darwin_386
github-release upload --user ad --repo gomgr --tag "$1" --name "gomgr_darwin_amd64" --file release/gomgr_darwin_amd64
github-release upload --user ad --repo gomgr --tag "$1" --name "gomgr_linux_386" --file release/gomgr_linux_386
github-release upload --user ad --repo gomgr --tag "$1" --name "gomgr_linux_amd64" --file release/gomgr_linux_amd64
github-release upload --user ad --repo gomgr --tag "$1" --name "gomgr_linux_arm" --file release/gomgr_linux_arm
