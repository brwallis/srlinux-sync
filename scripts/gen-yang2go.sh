#!/bin/bash
YANG_SRC_PATH="../appmgr"
GO_OUT_PATH="../internal/dssync/yang"

echo "Generate Go bindings for SR Linux DS synchronization YANG modules"
go mod download

YGOT_DIR=`go list -f '{{ .Dir }}' -m github.com/openconfig/ygot`

mkdir -p ${GO_OUT_PATH}
go run $YGOT_DIR/generator/generator.go \
   -path=${YANG_SRC_PATH}/ -output_file=${GO_OUT_PATH}/model.go -package_name=dssync_yang -generate_fakeroot \
   ${YANG_SRC_PATH}/dssync.yang

go mod tidy
