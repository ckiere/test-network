peer chaincode invoke -C auction --name blindauction --ctor '{"Args":["CreateAuction","'$1'", "'$2'"]}' -o localhost:7050 --peerAddresses localhost:7051 --peerAddresses localhost:9051
sleep $3
peer chaincode invoke -C auction --name blindauction --ctor '{"Args":["CloseAuction","'$1'"]}' -o localhost:7050 --peerAddresses localhost:7051 --peerAddresses localhost:9051
sleep $3
peer chaincode invoke -C auction --name blindauction --ctor '{"Args":["EndAuction","'$1'"]}' -o localhost:7050 --peerAddresses localhost:7051 --peerAddresses localhost:9051