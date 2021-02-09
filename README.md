# Network setup
- Generate certificates for organizations
cryptogen generate --config=./organizations/cryptogen/crypto-config-org1.yaml --output="organizations"
cryptogen generate --config=./organizations/cryptogen/crypto-config-org2.yaml --output="organizations"
cryptogen generate --config=./organizations/cryptogen/crypto-config-orderer.yaml --output="organizations"
- Create genesis block
configtxgen -profile TwoOrgsOrdererGenesis -channelID system-channel -outputBlock ./system-genesis-block/genesis.block -configPath configtx
- Launch network
docker-compose up
- Delete the whole network
docker-compose down -v
# Peer setup
- start CLI container (on windows)
docker run -it --rm --network="host" -v "%cd%":/mnt hyperledger/fabric-tools bash
source ./act_as_admin.sh org1 Org1MSP peer0 localhost:7051
- env
ORDERER_OPTS="-o localhost:7050 --tls --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"
# Channel creation
- Generate channel creation tx
configtxgen -profile TwoOrgsChannel -outputCreateChannelTx ./channel-artifacts/${CHANNEL_NAME}.tx -channelID $CHANNEL_NAME
- Generate anchor peer tx
configtxgen -profile TwoOrgsChannel -outputAnchorPeersUpdate ./channel-artifacts/Org1anchors.tx -channelID $CHANNEL_NAME -asOrg Org1
- Create channel
peer channel create -c $CHANNEL_NAME --ordererTLSHostnameOverride orderer.example.com -f ./channel-artifacts/${CHANNEL_NAME}.tx --outputBlock ./channel-artifacts/${CHANNEL_NAME}.block $ORDERER_OPTS
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
- Query
peer chaincode query -C channel1 --name test1 --ctor '{"Args":["QueryAuction","bla"]}'
- Invoke
peer chaincode invoke -C channel1 --name test1 --ctor '{"Args":["CreateAuction","bla", "test"]}' $ORDERER_OPTS --peerAddresses localhost:7051 --peerAddresses localhost:9051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt