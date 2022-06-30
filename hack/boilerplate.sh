#!/bin/bash
set -eu -o pipefail

for i in $(
  find ./pkg -name "*.go"
); do
  if ! grep -q "Tuan Anh Tran <me@tuananh.org>" $i; then
    cat hack/boilerplate.go.txt $i >$i.new && mv $i.new $i
  fi
done