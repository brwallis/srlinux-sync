#!/bin/sh

# Always exit on errors
set -e

# Set known directories
SRL_ETC_DIR="/host/etc/opt/srlinux"
DSSYNC_BASE_DIR="/host/"
DSSYNC_BIN_FILE="/dssync/bin/dssync"
DSSYNC_YANG="/dssync/yang/dssync.yang"
DSSYNC_CONFIG="/dssync/dssync_config.yml"

# Give help text for parameters.
usage()
{
    printf "This is an entrypoint script for the SR Linux DS sync agent to overlay its\n"
    printf "binary and configuration into the correct locations in a filesystem.\n"
    printf "\n"
    printf "./entrypoint.sh\n"
    printf "\t-h --help\n"
    printf "\t--srl-etc-dir=%s\n" $SRL_ETC_DIR
    printf "\t--dssync-bin-file=%s\n" $DSSYNC_BIN_FILE
    printf "\t--dssync-yang=%s\n" $DSSYNC_YANG
    printf "\t--dssync-config=%s\n" $DSSYNC_CONFIG
}

# Parse parameters given as arguments to this script.
while [ "$1" != "" ]; do
    PARAM=$(echo "$1" | awk -F= '{print $1}')
    VALUE=$(echo "$1" | awk -F= '{print $2}')
    case $PARAM in
        -h | --help)
            usage
            exit
            ;;
        --srl-etc-dir)
            SRL_ETC_DIR=$VALUE
            ;;
        --dssync-bin-file)
            DSSYNC_BIN_FILE=$VALUE
            ;;
        --dssync-yang)
            DSSYNC_YANG=$VALUE
            ;;
        --dssync-config)
            DSSYNC_CONFIG=$VALUE
            ;;
        *)
            /bin/echo "ERROR: unknown parameter \"$PARAM\""
            usage
            exit 1
            ;;
    esac
    shift
done


# Loop through and verify each location each.
for i in $SRL_ETC_DIR $DSSYNC_BIN_FILE $DSSYNC_YANG $DSSYNC_CONFIG
do
  if [ ! -e "$i" ]; then
    /bin/echo "Location $i does not exist"
    exit 1;
  fi
done

# Set up directories
mkdir -p ${SRL_ETC_DIR}/dssync/bin
mkdir -p ${SRL_ETC_DIR}/dssync/yang
mkdir -p ${SRL_ETC_DIR}/appmgr
# Add in the K8 service host/port, these are lost when app_mgr launches an application
# These env vars always exist when running inside K8
#echo "Kubernetes service host is: $KUBERNETES_SERVICE_HOST"
#echo "Kubernetes service port is: $KUBERNETES_SERVICE_PORT"
#echo "Updating $DSSYNC_CONFIG..."
#sed -i 's/$KUBERNETES_SERVICE_HOST/'"$KUBERNETES_SERVICE_HOST"'/' "$DSSYNC_CONFIG"
#sed -i 's/$KUBERNETES_SERVICE_PORT/'"$KUBERNETES_SERVICE_PORT"'/' "$DSSYNC_CONFIG"
#sed -i 's/$KUBERNETES_NODE_NAME/'"$KUBERNETES_NODE_NAME"'/' "$DSSYNC_CONFIG"
#sed -i 's/$KUBERNETES_NODE_IP/'"$KUBERNETES_NODE_IP"'/' "$DSSYNC_CONFIG"
# Copy files into proper places
cp -f "$DSSYNC_BIN_FILE" "$SRL_ETC_DIR/dssync/bin/"
cp -f "$DSSYNC_YANG" "$SRL_ETC_DIR/dssync/yang/"
cp -f "$DSSYNC_CONFIG" "$SRL_ETC_DIR/appmgr/"

echo "Entering sleep... (success)"

# Sleep forever. 
# sleep infinity is not available in alpine; instead lets go sleep for ~68 years. Hopefully that's enough sleep
sleep 2147483647
