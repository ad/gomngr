#!/bin/bash
echo "$1"
git tag "$1" && git push --tags
gox -output="release/{{.Dir}}_{{.OS}}_{{.Arch}}" -osarch="darwin/386 darwin/amd64 linux/386 linux/amd64 linux/arm"
github-release release --user ad --repo gomngr --tag "$1" --name "$1"
github-release upload --user ad --repo gomngr --tag "$1" --name "gomngr_darwin_386" --file release/gomngr_darwin_386
github-release upload --user ad --repo gomngr --tag "$1" --name "gomngr_darwin_amd64" --file release/gomngr_darwin_amd64
github-release upload --user ad --repo gomngr --tag "$1" --name "gomngr_linux_386" --file release/gomngr_linux_386
github-release upload --user ad --repo gomngr --tag "$1" --name "gomngr_linux_amd64" --file release/gomngr_linux_amd64
github-release upload --user ad --repo gomngr --tag "$1" --name "gomngr_linux_arm" --file release/gomngr_linux_arm
