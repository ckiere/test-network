# Network setup
- Generate certificates for organizations
cryptogen generate --config=./organizations/cryptogen/crypto-config-org1.yaml --output="organizations"
cryptogen generate --config=./organizations/cryptogen/crypto-config-org2.yaml --output="organizations"
cryptogen generate --config=./organizations/cryptogen/crypto-config-orderer.yaml --output="organizations"
- Create genesis block
configtxgen -profile TwoOrgsOrdererGenesis -channelID system-channel -outputBlock ./system-genesis-block/genesis.block -configPath configtx
- Launch network
docker-compose up
- on WSL, syncing the time is somteime needed
sudo hwclock -s
- Delete the whole network
docker-compose down -v
# Peer setup
- start CLI container (on windows)
docker run -it --rm --network="host" -v "%cd%":/mnt hyperledger/fabric-tools bash
docker run -it --rm --network="host" -v "%cd%":/mnt fabric-dac-tools:1.0 bash
source ./act_as_admin.sh org1 Org1MSP peer0 localhost:7051
- env
ORDERER_OPTS="-o localhost:7050 --tls --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"
# Fabric CA
export FABRIC_CA_CLIENT_HOME=/mnt/organizations/fabric-ca/idemix
export FABRIC_CA_CLIENT_TLS_CERTFILES=/mnt/organizations/fabric-ca/idemix/ca-cert.pem
fabric-ca-client enroll -d -u https://admin:adminpw@localhost:7054
fabric-ca-client register --id.name user1 --id.secret pswd --id.type client --id.affiliation org1 --id.attrs role=4 -u https://localhost:7054
fabric-ca-client enroll --enrollment.type idemix -u https://user1:pswd@localhost:7054
# Channel creation
- Generate channel creation tx
configtxgen -profile TwoOrgsChannel -outputCreateChannelTx ./channel-artifacts/${CHANNEL_NAME}.tx -channelID $CHANNEL_NAME -configPath configtx
- Generate anchor peer tx
configtxgen -profile TwoOrgsChannel -outputAnchorPeersUpdate ./channel-artifacts/Org1anchors.tx -channelID $CHANNEL_NAME -asOrg Org1
- Create channel
peer channel create -c $CHANNEL_NAME -f ./channel-artifacts/${CHANNEL_NAME}.tx --outputBlock ./channel-artifacts/${CHANNEL_NAME}.block -o localhost:7050
- Join channel
peer channel join --blockpath ./channel-artifacts/${CHANNEL_NAME}.block
# Chaincode
- Package chaincode
peer lifecycle chaincode package test1.tar.gz --path chaincode --lang golang --label test1
- Install chaincode
peer lifecycle chaincode install test1.tar.gz
- Approve
peer lifecycle chaincode approveformyorg --channelID ${CHANNEL_NAME} --name test1 --version 1.0 --package-id test1:70ef09105a883fa88c23fbf503837c2bdef69971095a14bc0eeb9c39b5b8d3da --sequence 1 $ORDERER_OPTS
- Commit
peer lifecycle chaincode commit --channelID ${CHANNEL_NAME} --name test1 --version 1.0  --sequence 1 $ORDERER_OPTS --peerAddresses localhost:7051 --peerAddresses localhost:9051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
- Commit (without TLS)
peer lifecycle chaincode commit --channelID ${CHANNEL_NAME} --name test1 --version 1.0  --sequence 1 -o localhost:7050 --peerAddresses localhost:7051 --peerAddresses localhost:9051
- Query
peer chaincode query -C channel1 --name test1 --ctor '{"Args":["QueryAuction","bla"]}'
- Invoke
peer chaincode invoke -C channel1 --name test1 --ctor '{"Args":["CreateAuction","bla", "test"]}' $ORDERER_OPTS --peerAddresses localhost:7051 --peerAddresses localhost:9051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt

Without TLS:
peer chaincode query -C auction --name blindauction --ctor '{"Args":["QueryAuction","testauction"]}'