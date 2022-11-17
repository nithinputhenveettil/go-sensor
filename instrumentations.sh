#!/bin/sh

# This script provides utilities to update all instrumentations to refer to the latest core module.
#
# Usage
#
# Update instrumentation phase
#
# 1. Run `./instrumentations.sh update`. This will update all instrumentations to reference the latest core module.
# 2. Run `make test` to ensure that all instrumentations work with the latest core.
# 3. If any errors are detected in some instrumentation, fix them.
# 4. If everything is fine, commit your changes, open a PR and get it merged into the master branch.
#
# Release phase
# 1. Run `./instrumentations.sh release` to create tags for each instrumentation with a new minor version, and update all version.go files.

set -eo pipefail

# Checks if gh is installed, otherwise stop the script
if ! [ -x "$(command -v gh)" ]; then
  echo 'Error: gh is not installed.' >&2
  exit 1
fi

# Checks if the user is logged into Github, otherwise stop the script
if gh auth status 2>&1 | grep -i "You are not logged"; then
  echo "Error: You must log into Github"
  exit 1
fi

# CORE_VERSION=latest
CORE_VERSION=v1.45.0

# List of instrumentation folders
LIB_LIST=$(find ./instrumentation -name go.mod -exec dirname {} \; | grep -v "/instasarama/example")

# Updates all instrumentations to use the @latest version of the core module
run_update() {
  for lib in $LIB_LIST
    do cd $lib && go get github.com/instana/go-sensor@$CORE_VERSION && go mod tidy && cd -;
  done
}

# Updates version.go and creates a new tag for every instrumentation, incrementing the minor version
run_release() {
  TAGS=""
  for lib in $LIB_LIST
    do LIB_PATH="$(echo $lib | sed 's/\.\///')"
    VERSION=$(git tag -l "$LIB_PATH*" | sort -V | tail -n1 | sed "s/.*v//")

    if [ -z $VERSION ]; then
      VERSION="0.0.0"
    fi

    MINOR_VERSION=$(echo $VERSION | sed -En 's/[0-9]+\.([0-9]+)\.[0-9]+/\1/p')
    MAJOR_VERSION=$(echo $VERSION | sed -En 's/([0-9]+)\.[0-9]+\.[0-9]+/\1/p')
    MINOR_VERSION=$(($MINOR_VERSION+1))
    NEW_VERSION="$MAJOR_VERSION.$MINOR_VERSION.0"

    # Updates the minor version in version.go
    sed -i '' -E "s/[0-9]+\.[0-9]+\.[0-9]+/${NEW_VERSION}/" $lib/version.go | tail -1

    # Tags to be created after version.go is merged to the master branch with the new version
    TAGS="$TAGS $LIB_PATH@v$MAJOR_VERSION.$MINOR_VERSION.0"
  done

  # Commit all version.go files to the master branch
  git add ./instrumentation/**/version.go
  git add ./instrumentation/**/**/version.go
  git commit -m "Bumping new version of the instrumentation"
  git push origin master

  echo "Creating tags for each instrumentation"

  for t in $TAGS
    do git tag $t && git push origin $t
  done

  # Release every instrumentation
  for t in $TAGS
    do gh release create $t \
		--title $t \
		--notes "Update instrumentations to the latest core module"
  done
}

if [ "$1" = "update" ]; then
  run_update
fi

if [ "$1" = "release" ]; then
  run_release
fi
