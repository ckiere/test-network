package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ckiere/test-network/client-dac-go/crypto"
	"github.com/ckiere/test-network/client-dac-go/dacidentity"
	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"io/ioutil"
	"os"
	"strconv"
	"time"
)

const configFileName = "DacConfig.json"
const skFileName = "DacSk.json"
const channelName = "auction"
const chaincodeID = "blindauction"

func main() {
	argc := len(os.Args)

	if argc > 1 {
		cmd := os.Args[1]
		if cmd == "createconfig" {
			createConfigFiles()
		} else if cmd == "createauthority" {
			if argc == 3 {
				createAuthorityFiles(os.Args[2])
			} else {
				fmt.Println("Wrong number of arguments")
			}
		} else if cmd == "createidentity" {
			if argc == 4 {
				createIdentityFiles(os.Args[2], os.Args[3])
			} else {
				fmt.Println("Wrong number of arguments")
			}
		} else if cmd == "client" {
			if argc > 5 {
				price, err := strconv.Atoi(os.Args[4])
				if err == nil && price > 0 {
					launchClient(os.Args[2], os.Args[3], price, os.Args[5:])
				} else {
					fmt.Println("Invalid price")
				}
			} else {
				fmt.Println("Wrong number of arguments")
			}
		} else {
			fmt.Println("Unknown command")
		}
	} else {
		fmt.Println("No command given as argument")
	}
}

func launchClient(username string, auctionID string, price int, endpoints []string) {
	configBytes, _ := ioutil.ReadFile(configFileName)
	dacConfig, err := dacidentity.CreateConfigFromBytes(configBytes)
	userConfigBytes, _ := ioutil.ReadFile(username + ".json")
	var userConfig dacidentity.CredentialsConfig
	json.Unmarshal(userConfigBytes, &userConfig)
	user, err := dacidentity.CreateUser(*dacConfig, userConfig, username, "DacMSP")

	sdk, err := fabsdk.New(config.FromFile("connection-org1.yaml"), fabsdk.WithCorePkg(dacidentity.NewProviderFactory()))
	if err != nil {
		panic(err)
	}

	dacClientChannelContext := sdk.ChannelContext(channelName, fabsdk.WithIdentity(user))
	client, err := channel.New(dacClientChannelContext)
	if err != nil {
		panic(err)
	}

	// get the auctioneer public key to encrypt the bid
	response, err := client.Query(channel.Request{ChaincodeID: chaincodeID, Fcn: "QueryAuctioneerPk", Args: [][]byte{[]byte(auctionID)}},
		channel.WithRetry(retry.DefaultChannelOpts), channel.WithTargetEndpoints(endpoints...))
	if err != nil {
		panic(err)
	}

	var auctioneerPk [32]byte
	auctioneerPkBytes := make([]byte, base64.StdEncoding.DecodedLen(len(response.Payload)))
	_, err = base64.StdEncoding.Decode(auctioneerPkBytes, response.Payload)
	if err != nil {
		panic(err)
	}
	copy(auctioneerPk[:], auctioneerPkBytes)

	// commit to a bid and prove knowledge of opening values
	com, r, err := crypto.Commit(price)
	if err != nil {
		panic(err)
	}
	comBytes := com.Marshal()
	t, s1, s2, err := crypto.ProveCommit(price, r, comBytes, nil)
	if err != nil {
		panic(err)
	}
	proofBytes := crypto.CommitProofToBytes(t, s1, s2)

	comBase64 := base64.StdEncoding.EncodeToString(comBytes)
	proofBase64 := base64.StdEncoding.EncodeToString(proofBytes)
	response, _ = client.Execute(channel.Request{ChaincodeID: chaincodeID, Fcn: "SendCommitment", Args: [][]byte{[]byte(auctionID), []byte(comBase64), []byte(proofBase64)}},
		channel.WithRetry(retry.DefaultChannelOpts), channel.WithTargetEndpoints(endpoints...))
	txID := response.Payload

	// renew the nym identity
	//user.UpdateNymIdentity()

	// pause to wait for the second phase of the auction
	time.Sleep(time.Duration(30) * time.Second)

	// encrypt the bid
	encryptedBid, err := crypto.Encrypt(price, r, &auctioneerPk)
	if err != nil {
		panic(err)
	}

	// generate proof of knowledge of opening values
	t, s1, s2, err = crypto.ProveCommit(price, r, comBytes, encryptedBid)
	if err != nil {
		panic(err)
	}
	proof2Bytes := crypto.CommitProofToBytes(t, s1, s2)

	// reveal the encrypted bid
	encryptedBidBase64 := base64.StdEncoding.EncodeToString(encryptedBid)
	proof2Base64 := base64.StdEncoding.EncodeToString(proof2Bytes)
	client.Execute(channel.Request{ChaincodeID: chaincodeID, Fcn: "RevealBid", Args: [][]byte{[]byte(auctionID), txID,
		[]byte(""), []byte(encryptedBidBase64), []byte(proof2Base64)}},
		channel.WithRetry(retry.DefaultChannelOpts), channel.WithTargetEndpoints(endpoints...))
}

func createConfigFiles() {
	dacConfig, rootSk := dacidentity.CreateConfig()
	configBytes, _ := json.Marshal(dacConfig)
	skBytes := make([]byte, FP256BN.MODBYTES)
	rootSk.ToBytes(skBytes)
	ioutil.WriteFile(configFileName, configBytes, 0644)
	ioutil.WriteFile(skFileName, skBytes, 0644)
}

func createAuthorityFiles(authName string) {
	prg := dacidentity.NewRand()
	configBytes, _ := ioutil.ReadFile(configFileName)
	rootSkBytes, _ := ioutil.ReadFile(skFileName)
	dacConfig, _ := dacidentity.CreateConfigFromBytes(configBytes)
	rootPk, _ := dacConfig.RootPk()
	rootSk := FP256BN.FromBytes(rootSkBytes)
	ys, _ := dacConfig.Ys()
	authConfig := createAuthority(rootPk, rootSk, prg, ys)
	authConfigBytes, _ := json.Marshal(authConfig)
	ioutil.WriteFile(authName + ".json", authConfigBytes, 0700)
}

func createIdentityFiles(authName string, idName string) {
	prg := dacidentity.NewRand()
	configBytes, _ := ioutil.ReadFile(configFileName)
	dacConfig, _ := dacidentity.CreateConfigFromBytes(configBytes)
	authConfigBytes, _ := ioutil.ReadFile(authName + ".json")
	var authConfig dacidentity.CredentialsConfig
	json.Unmarshal(authConfigBytes, &authConfig)
	authCreds := *dac.CredentialsFromBytes(authConfig.CredentialsBytes)
	authSk := FP256BN.FromBytes(authConfig.SkBytes)
	ys, _ := dacConfig.Ys()
	idConfig := createIdentity(authCreds, authSk, prg, ys)
	idConfigBytes, _ := json.Marshal(idConfig)
	ioutil.WriteFile(idName + ".json", idConfigBytes, 0700)
}

func createAuthority(rootPk dac.PK, rootSk dac.SK, prg *amcl.RAND, ys [][]interface{}) dacidentity.CredentialsConfig {
	authSk, authPk := dac.GenerateKeys(prg, 1)
	rootCreds := dac.MakeCredentials(rootPk)
	rootCreds.Delegate(rootSk, authPk, make([]interface{}, 0), prg, ys)
	credsBytes := rootCreds.ToBytes()
	authSkBytes := make([]byte, FP256BN.MODBYTES)
	authSk.ToBytes(authSkBytes)
	authPkBytes := dac.PointToBytes(authPk)
	return dacidentity.CredentialsConfig{CredentialsBytes: credsBytes, SkBytes: authSkBytes, PkBytes: authPkBytes}
}

func createIdentity(authCreds dac.Credentials, authSk dac.SK, prg *amcl.RAND, ys [][]interface{}) dacidentity.CredentialsConfig {
	idSk, idPk := dac.GenerateKeys(prg, 2)
	authCreds.Delegate(authSk, idPk, make([]interface{}, 0), prg, ys)
	credsBytes := authCreds.ToBytes()
	idSkBytes := make([]byte, FP256BN.MODBYTES)
	idSk.ToBytes(idSkBytes)
	idPkBytes := dac.PointToBytes(idPk)
	return dacidentity.CredentialsConfig{CredentialsBytes: credsBytes, SkBytes: idSkBytes, PkBytes: idPkBytes}
}
