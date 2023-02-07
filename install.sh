#!/bin/bash
# Copyright MontFerret Team 2020
# Licensed under the MIT license.
projectName="MontFerret"
appName="cli"
binName="ferret"
fullAppName="${projectName} $(fn_echo ${appName} | awk '{print toupper(substr($0,0,1)) substr($0,2)}')"
defaultLocation="${HOME}/.ferret"
defaultVersion="latest"
location=${LOCATION:-$defaultLocation}
version=${VERSION:-$defaultVersion}

if [ "$version" = "$defaultVersion" ]; then
    version=$(curl -sI https://github.com/${projectName}/${appName}/releases/latest | awk '{print tolower($0)}' | grep location: | awk -F"/" '{ printf "%s", $NF }' | tr -d '\r')
fi

baseUrl=https://github.com/${projectName}/$appName/releases/download/$version

report() {
  command printf %s\\n "$*" 2>/dev/null
}

detectProfile() {
  if [ "${PROFILE-}" = '/dev/null' ]; then
    # the user has specifically requested NOT to have nvm touch their profile
    return
  fi

  if [ -n "${PROFILE}" ] && [ -f "${PROFILE}" ]; then
    report "${PROFILE}"
    return
  fi

  local DETECTED_PROFILE
  DETECTED_PROFILE=''

  if [ "${SHELL#*bash}" != "$SHELL" ]; then
    if [ -f "$HOME/.bashrc" ]; then
      DETECTED_PROFILE="$HOME/.bashrc"
    elif [ -f "$HOME/.bash_profile" ]; then
      DETECTED_PROFILE="$HOME/.bash_profile"
    fi
  elif [ "${SHELL#*zsh}" != "$SHELL" ]; then
    if [ -f "$HOME/.zshrc" ]; then
      DETECTED_PROFILE="$HOME/.zshrc"
    fi
  fi

  if [ -z "$DETECTED_PROFILE" ]; then
    for EACH_PROFILE in ".profile" ".bashrc" ".bash_profile" ".zshrc"
    do
      if DETECTED_PROFILE="$(check_path "${HOME}/${EACH_PROFILE}")"; then
        break
      fi
    done
  fi

  if [ -n "$DETECTED_PROFILE" ]; then
    report "$DETECTED_PROFILE"
  fi
}

updateProfile() {
  profile=$(detectProfile)

  if [[ ":$PATH:" == *":$location:"* ]]; then
    return
  fi

  report "Updating profile $profile"

  if [ -z "$profile" ]; then
    report "Profile not found. Tried ${DETECTED_PROFILE-} (as defined in \$PROFILE), ~/.bashrc, ~/.bash_profile, ~/.zshrc, and ~/.profile."
    report "Append the following lines to the correct file yourself:"
    report
    report "export PATH=\$PATH:${location}"
    report
  else
    if ! grep -qc "${location}" "$profile"; then
      report "export PATH=\$PATH:${location}" >> "$profile"
    fi
  fi
}

checkHash(){
    downloadDir=$1
    sha_cmd="sha256sum"

    if [ ! -x "$(command -v ${sha_cmd})" ]; then
        sha_cmd="shasum -a 256"
    fi

    if [ -x "$(command -v "${sha_cmd}")" ]; then

    (cd "${downloadDir}" && curl -sSL "${baseUrl}"/lab_checksums.txt | $sha_cmd -c >/dev/null)
        if [ "$?" != "0" ]; then
            # rm $downloadFile
            report "Binary checksum didn't match. Exiting"
            exit 1
        fi
    fi
}

getPlatformSuffix() {
  local platformName=$(uname)
  local userid=$(id -u)

  local platform=""
  case $platformName in
  "Darwin")
  platform="_darwin"
  ;;
  "Linux")
  platform="_linux"
  ;;
  "Windows")
  platform="_windows"
  ;;
  esac

  if [ "$platform" = "" ]; then
      report "$platformName is not supported. Exiting..."
      exit 1
  fi

  local archName=$(uname -m)
  local arch=""
  case $archName in
  "x86_64")
  arch="_x86_64"
  ;;
  "aarch64")
  arch="_arm64"
  ;;
  "arm64")
  arch="_arm64"
  ;;
  esac

  if [ "${arch}" = "" ]; then
      report "${archName} is not supported. Exiting..."
      exit 1
  fi

  echo "${platform}${arch}"
}

installPackage() {
  report "Installing ${fullAppName} ${version}..."
  # Download the archive to a temporary location
  local tmpDir=$(mktemp -d -t "${projectName}.${appName}")
  local suffix=$(getPlatformSuffix)
  local fileName="${appName}${suffix}"
  local downloadDir="${tmpDir}/${fileName}@${version}"

  if [ ! -d "${downloadDir}" ]; then
      mkdir "${downloadDir}"

      if [ $? -ne 0 ]; then
        report "Can't create temp directory. Exiting..."
        exit 1
     fi
  fi

  local downloadFile="${downloadDir}/${fileName}.tar.gz"
  local url="$baseUrl/${fileName}.tar.gz"
  report "Downloading package $url as $downloadFile"

  curl -sSL "${url}" | tar xz --directory "${downloadDir}"

  if [ $? -ne 0 ]; then
      report "Failed to download file. Exiting..."
      exit 1
  fi

  report "Download complete."

  checkHash "${downloadDir}"

  if [ $? -ne 0 ]; then
      report "Failed to check hash. Exiting..."
      exit 1
  fi

  local downloadedFile="${downloadDir}/${binName}"

  if [ ! -d "${location}" ]; then
      mkdir "${location}"

      if [ $? -ne 0 ]; then
        report "Can't create installation directory. Exiting..."
        exit 1
      fi
  fi

  report "Copying ${downloadedFile} to ${location}"
  report

  cp "${downloadedFile}" "${location}"

  if [ "$?" != "0" ]; then
      report "Failed to copy file. Exiting..."
      exit 1
  fi

  if [ -d "${downloadDir}" ]; then
      rm -rf "${downloadDir}"
  fi

  local executable="$location/$binName"

  chmod +x "${executable}"

  updateProfile "${location}"

  report "New version of ${fullAppName} installed to ${location}"

  "$executable" version
}

installPackage
