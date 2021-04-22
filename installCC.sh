CHANNEL_NAME="auction"
ORDERER="localhost:7050"
PEERS="--peerAddresses localhost:7051 --peerAddresses localhost:9051"
VER="$1"
SEQ="$2"

peer lifecycle chaincode package blindauction.tar.gz --path auction-chaincode --lang golang --label blindauction
PKGID=blindauction:$(sha256sum blindauction.tar.gz | cut -d " " -f 1)

source ./act_as_admin.sh org1 Org1MSP peer0 localhost:7051
peer lifecycle chaincode install blindauction.tar.gz
peer lifecycle chaincode approveformyorg --channelID $CHANNEL_NAME --name blindauction --version $VER --package-id $PKGID --sequence $SEQ -o $ORDERER
source ./act_as_admin.sh org2 Org2MSP peer0 localhost:9051
peer lifecycle chaincode install blindauction.tar.gz
peer lifecycle chaincode approveformyorg --channelID $CHANNEL_NAME --name blindauction --version $VER --package-id $PKGID --sequence $SEQ -o $ORDERER
peer lifecycle chaincode commit --channelID ${CHANNEL_NAME} --name blindauction --version $VER  --sequence $SEQ -o $ORDERER $PEERS