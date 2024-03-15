#!/usr/bin/env bash

set -e

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

PLATFORM=
ARCH=
VERSION=
RELEASES_URL="https://github.com/plandex-ai/plandex/releases/download"

# Set platform
case "$(uname -s)" in
 Darwin)
   PLATFORM='darwin'
   ;;

 Linux)
   PLATFORM='linux'
   ;;

 FreeBSD)
   PLATFORM='freebsd'
   ;;

 CYGWIN*|MINGW*|MSYS*)
   PLATFORM='windows'
   ;;

 *)
   echo "Platform may or may not be supported. Will attempt to install."
   PLATFORM='linux'
   ;;
esac

# Set arch
if [[ "$(uname -m)" == 'x86_64' ]]; then
  ARCH="amd64"
elif [[ "$(uname -m)" == 'arm64' || "$(uname -m)" == 'aarch64' ]]; then
  ARCH="arm64"
fi

if [[ "$(cat /proc/1/cgroup 2> /dev/null | grep docker | wc -l)" > 0 ]] || [ -f /.dockerenv ]; then
  IS_DOCKER=true
else
  IS_DOCKER=false
fi

# Set Version
if [[ -z "${PLANDEX_VERSION}" ]]; then  
  VERSION=$(curl -sL https://plandex.ai/cli-version.txt)
else
  VERSION=$PLANDEX_VERSION
  echo "Using custom version $VERSION"
fi


welcome_plandex () {
  echo "Plandex $VERSION Quick Install"
  echo "Copyright (c) 2024 Plandex Inc."
  echo ""
}

cleanup () {
  echo "Cleaning up..."
  cd "${SCRIPT_DIR}"
  rm -rf plandex_install_tmp
}

download_plandex () {
  ENCODED_TAG="cli%2Fv${VERSION}"

  url="${RELEASES_URL}/${ENCODED_TAG}/plandex_${VERSION}_${PLATFORM}_${ARCH}.tar.gz"

  mkdir plandex_install_tmp
  cd plandex_install_tmp

  echo "Downloading Plandex tarball from $url"
  curl -s -L -o plandex.tar.gz "${url}"

  tar zxf plandex.tar.gz 1> /dev/null

  if [ "$PLATFORM" == "darwin" ] || $IS_DOCKER ; then
    if [[ -d /usr/local/bin ]]; then
      mv plandex /usr/local/bin/
      echo "Plandex is installed in /usr/local/bin"
    else
      echo >&2 'Error: /usr/local/bin does not exist. Create this directory with appropriate permissions, then re-install.'
      cleanup
      exit 1
    fi
  elif [ "$PLATFORM" == "windows" ]; then
    # ensure $HOME/bin exists (it's in PATH but not present in default git-bash install)
    mkdir "$HOME/bin" 2> /dev/null
    mv plandex.exe "$HOME/bin/"
    echo "Plandex is installed in '$HOME/bin'"
  else
    if [ $UID -eq 0 ]
    then
      # we are root
      mv plandex /usr/local/bin/  
    elif hash sudo 2>/dev/null;
    then
      # not root, but can sudo
      sudo mv plandex /usr/local/bin/
    else
      echo "ERROR: This script must be run as root or be able to sudo to complete the installation."
      exit 1
    fi
    
    echo "Plandex is installed in /usr/local/bin"
  fi  

  # create 'pdx' alias, but don't ovewrite existing pdx command
  if [ ! -x "$(command -v pdx)" ]; then
    echo "creating pdx alias"
    LOC=$(which plandex)
    BIN_DIR=$(dirname $LOC)
    error_msg=$(ln -s "$LOC" "$BIN_DIR/pdx" 2>&1) || { echo "Failed to create 'pdx' alias for Plandex. Error: $error_msg. Please create it manually if needed."; }
  fi
}

welcome_plandex
download_plandex
cleanup

echo "Installation complete. Info:"
echo ""
plandex -h

