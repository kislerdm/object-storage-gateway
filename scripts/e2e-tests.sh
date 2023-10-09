#!/usr/bin/env bash

URL="http://localhost:3000/object"
URL_TOFU="https://github.com/opentofu/opentofu/releases/download/v1.6.0-alpha1/tofu_1.6.0-alpha1_darwin_arm64.zip"

echo "Init"

if [ ! -d ${PWD}/samples ]; then mkdir -p ${PWD}/samples/get; fi

function deleteSamples() {
    cd .. && rm -r ${PWD}/samples
}

cd samples || exit 1

echo "generate dummy file tinytextfile.txt"
echo "foo bar baz" >> tinytextfile.txt

echo "download test sample from ${URL_TOFU} to tofu.zip"
curl -s -Lo tofu.zip ${URL_TOFU}
if [ $? -gt 0 ]; then echo "error" && deleteSamples; exit 1; fi

echo "unzip tofu.zip"
unzip -qq tofu.zip
if [ $? -gt 0 ]; then echo "error" && deleteSamples; exit 1; fi

echo "Run tests"

files=( tinytextfile.txt LICENSE tofu.zip )
objects=( tinytextfile LICENSE tofuzip )

for i in "${!files[@]}"; do

  fileName=${files[$i]}
  objectID=${objects[$i]}

  echo "upload ${fileName} as objectID ${objectID}"
  curl -s -T ${fileName} ${URL}/${objectID}

  echo "download ${objectID} to ./get/${fileName}"
  curl -s -o ./get/${fileName} ${URL}/${objectID}

  if [ "$(grep -e "error" ./get/${fileName} | wc -l)" -gt 0 ]; then
    echo "downloading error"
    deleteSamples
    exit 1
  fi

  echo "compare files. want: ${fileName}, got: ./get/${fileName}."

  if [ "$(diff "./get/${fileName}" "${fileName}" | wc -l)" -gt 0 ]; then
    echo "FAIL"
    deleteSamples
    exit 1
  fi

  echo "OK"

done

deleteSamples

echo "Successfully Completed"
