#!/usr/bin/env bash
# Helper script to ease publishing new releases of meguca
# Usage ./scripts/release.sh <version>
# Example: ./scripts/release.sh v1.7.2

version=$1

npm version $version --no-git-tag-version
sed -i "s/##vNext/##${version} - $(date +%Y-%m-%d)/" CHANGELOG.md
git commit CHANGELOG.md package.json -m $version
git tag $version
git push
git push origin $version
