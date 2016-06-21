#!/usr/bin/env bash
# Helper script to ease publishing new releases of meguca
# Usage ./scripts/release.sh <version>
# Example: ./scripts/release.sh v1.7.2

version=$1

sed -i "s/##vNext/##${version} - $(date +%Y-%m-%d)/" CHANGELOG.md
git commit CHANGELOG.md -m $version || exit 1
git tag $version || exit 1
git push || exit 1
git push origin $version || exit 1
make package
