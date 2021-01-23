ORG=$1
MSPID=$2
PEER=$3
ADDR=$4
DOMAIN="example.com"

export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID=${MSPID}
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/${ORG}.${DOMAIN}/peers/${PEER}.${ORG}.${DOMAIN}/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/${ORG}.${DOMAIN}/users/Admin@${ORG}.${DOMAIN}/msp
export CORE_PEER_ADDRESS=${ADDR}