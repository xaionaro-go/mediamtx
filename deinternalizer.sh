#!/bin/sh -x
git checkout main
git reset --hard 54c07370744538914078e1497015a95b9a1ae046
git pull https://github.com/bluenviron/mediamtx
mv internal pkg
find . -type f -name '*.go' -exec sed -e 's%mediamtx/internal/%mediamtx/pkg/%g' -e 's%bluenviron/mediamtx/%xaionaro-go/mediamtx/%g' -i {} +
sed -e 's%bluenviron/mediamtx%xaionaro-go/mediamtx%g' -i go.mod go.sum
go fmt ./... >/dev/null
git add .
git commit -a -m 'De-internalized'
exit 0
