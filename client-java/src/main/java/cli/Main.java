package cli;

import client.Client;
import org.hyperledger.fabric.sdk.Enrollment;
import org.hyperledger.fabric.sdk.NetworkConfig;
import org.hyperledger.fabric.sdk.User;
import org.hyperledger.fabric.sdk.exception.InvalidArgumentException;
import org.hyperledger.fabric.sdk.exception.NetworkConfigurationException;
import org.hyperledger.fabric.sdk.exception.ProposalException;
import utils.ClientUser;
import utils.IdemixEnrollmentLoader;
import utils.X509EnrollmentLoader;

import java.io.File;
import java.io.IOException;
import java.security.NoSuchAlgorithmException;
import java.security.spec.InvalidKeySpecException;
import java.util.concurrent.ExecutionException;

public class Main {
    public static void main(String[] args) throws IOException, NetworkConfigurationException, InvalidKeySpecException, NoSuchAlgorithmException, ProposalException, InvalidArgumentException, ExecutionException, InterruptedException {
        String basePath = "../organizations/peerOrganizations/idemix.example.com/";
        File configFile = new File(basePath + "msp/user/SignerConfig");
        File ipkFile = new File(basePath + "msp/IssuerPublicKey");
        File rpkFile = new File(basePath + "msp/RevocationPublicKey");
        File networkConfigFile = new File(basePath + "connection-org1.yaml");
        String userName = "user1";
        String mspId = "IdemixMSP";
        Enrollment enrollment = IdemixEnrollmentLoader.loadFromFile(configFile, ipkFile, rpkFile, mspId, 4);
        User user = new ClientUser(userName, mspId, enrollment);
        NetworkConfig networkConfig = NetworkConfig.fromYamlFile(networkConfigFile);
        Client client = new Client(user, "idemixchannel", "smart1", networkConfig);
        client.sendTransaction("Store", new String[]{"key", "test"});
    }
}
