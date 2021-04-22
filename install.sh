#!/bin/bash

projectName="cli"
binName="ferret"
name="Ferret CLI"
defaultLocation="/usr/local/bin"
defaultVersion="latest"
location=${LOCATION:-$defaultLocation}
version=${VERSION:-$defaultVersion}

# Copyright MontFerret Team 2020
version=$(curl -sI https://github.com/MontFerret/$projectName/releases/latest | awk '{print tolower($0)}' | grep location: | awk -F"/" '{ printf "%s", $NF }' | tr -d '\r')
echo "Installing $name $version to $location"

if [ ! $version ]; then
    echo "Failed while attempting to install $name. Please manually install:"
    echo ""
    echo "1. Open your web browser and go to https://github.com/MontFerret/$projectName/releases"
    echo "2. Download the latest release for your platform."
    echo "3. chmod +x ./$projectName"
    echo "4. mv ./$projectName $location"
    exit 1
fi

checkHash(){
    sha_cmd="sha256sum"

    if [ ! -x "$(command -v $sha_cmd)" ]; then
        sha_cmd="shasum -a 256"
    fi

    if [ -x "$(command -v $sha_cmd)" ]; then

    (cd $targetDir && curl -sSL $baseUrl/$projectName_checksums.txt | $sha_cmd -c >/dev/null)
        if [ "$?" != "0" ]; then
            # rm $targetFile
            echo "Binary checksum didn't match. Exiting"
            exit 1
        fi
    fi
}

getPackage() {
    uname=$(uname)
    userid=$(id -u)

    platform=""
    case $uname in
    "Darwin")
    platform="_darwin"
    ;;
    "Linux")
    platform="_linux"
    ;;
    esac

    uname=$(uname -m)
    arch=""
    case $uname in
    "x86_64")
    arch="_x86_64"
    ;;
    esac
    case $uname in
    "aarch64")
    arch="_arm64"
    ;;
    esac

    if [ "$arch" = "" ]; then
        echo "$arch is not supported. Exiting"
        exit 1
    fi

    suffix=$platform$arch
    targetDir="/tmp/$projectName$suffix"

    if [ "$userid" != "0" ]; then
        targetDir="$(pwd)/$projectName$suffix"
    fi

    if [ ! -d $targetDir ]; then
        mkdir $targetDir
    fi

    targetFile="$targetDir/$binName"

    if [ -e $targetFile ]; then
        rm $targetFile
    fi

    echo

    if [ $location = $defaultLocation ]; then
        if [ "$userid" != "0" ]; then
            echo
            echo "========================================================="
            echo "==    As the script was run as a non-root user the     =="
            echo "==    following commands may need to be run manually   =="
            echo "========================================================="
            echo
            echo "  sudo cp $targetFile $location/$binName"
            echo "  rm -rf $targetDir"
            echo

            exit 1
        fi
    fi

    if [ ! -d $location ]; then
        mkdir $location
    fi

    baseUrl=https://github.com/MontFerret/$projectName/releases/download/$version
    url=$baseUrl/$projectName$suffix.tar.gz
    echo "Downloading package $url as $targetFile"

    curl -sSL $url | tar xz -C $targetDir

    if [ "$?" != "0" ]; then
        echo "Failed to download file"
        exit 1
    fi

    # checkHash

    chmod +x $targetFile

    echo "Download complete."
    echo
    echo "Attempting to move $targetFile to $location"

    mv $targetFile "$location/$binName"

    if [ "$?" = "0" ]; then
        echo "New version of $name installed to $location"
    fi

    if [ -d $targetDir ]; then
        rm -rf $targetDir
    fi

    "$location/$binName" version
}

getPackage