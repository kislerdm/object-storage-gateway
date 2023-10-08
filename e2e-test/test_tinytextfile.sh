#!/usr/bin/env sh

URL="http://localhost:3000/object"

echo "upload ./e2e-test/tinytextfile.txt as objectID tinytextfile"
curl -X PUT -d @${PWD}/e2e-test/tinytextfile.txt ${URL}/tinytextfile

echo "download tinytextfile to ./e2e-test/tinytextfile.txt-got"
curl -X GET -o ${PWD}/e2e-test/tinytextfile.txt-got ${URL}/tinytextfile

echo "compare files"

# shellcheck disable=SC2046
if [ $(diff "${PWD}/e2e-test/tinytextfile.txt-got" "${PWD}/e2e-test/tinytextfile.txt" | wc -l) -gt 0 ]; then
  echo Failed
  exit 1
fi
echo OK

rm "${PWD}/e2e-test/tinytextfile.txt-got"
