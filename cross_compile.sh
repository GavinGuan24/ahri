#!/usr/bin/env bash

cd $GOPATH/src/github.com/GavinGuan24/ahri/
rm -rf releases
mkdir releases

cd ./product/version
version=`go run version.go`
cd $GOPATH/src/github.com/GavinGuan24/ahri/

# $1 os, $2 arch
function build() {
  echo "[Building] OS: $1 , ARCH: $2"
  cd ./product/client
  CGO_ENABLED=0 GOOS=$1 GOARCH=$2 go build -o ahri-client
  mv ./ahri-client ../../releases
  cp ./ahri.hosts ../../releases
  cd $GOPATH/src/github.com/GavinGuan24/ahri/

  cd ./product/server
  CGO_ENABLED=0 GOOS=$1 GOARCH=$2 go build -o ahri-server
  mv ./ahri-server ../../releases
  cp ./gen_rsa_keys.sh ../../releases
  cd $GOPATH/src/github.com/GavinGuan24/ahri/

  cd ./releases
  tar zcf "ahri_"$version"_"$1"_"$2".tgz" ./ahri-client ./ahri.hosts ./ahri-server ./gen_rsa_keys.sh
  rm -rf ./ahri-client ./ahri.hosts ./ahri-server ./gen_rsa_keys.sh
  cd $GOPATH/src/github.com/GavinGuan24/ahri/
  echo "[OK] OS: $1 , ARCH: $2"
  echo "----------------------------"
  echo
}

echo "current version: $version"
echo ""

build windows 386
build windows amd64

build linux 386
build linux amd64
build linux arm
build linux arm64

build darwin 386
build darwin amd64

build freebsd 386
build freebsd amd64
build netbsd 386
build netbsd amd64
build openbsd 386
build openbsd amd64

tar zcf ./releases/"ahri_"$version"_src.tgz" ./core ./product ./test ./cross_compile.sh

echo "[Finished]"

exit 0
