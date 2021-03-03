package client;

import org.hyperledger.fabric.sdk.*;
import org.hyperledger.fabric.sdk.exception.*;
import org.hyperledger.fabric.sdk.security.CryptoSuite;
import org.hyperledger.fabric.sdk.security.CryptoSuiteFactory;

import java.lang.reflect.InvocationTargetException;
import java.util.Collection;
import java.util.concurrent.CompletableFuture;

public class Client {
    private HFClient client;
    private NetworkConfig networkConfig;
    private Channel channel;
    private ChaincodeID ccid;
    private int proposalWaitTime = 1000;

    public Client(User user, String channelName, String chaincodeName, NetworkConfig networkConfig) {
        this.networkConfig = networkConfig;
        setChaincode(chaincodeName);
        // Initialize the client
        client = HFClient.createNewInstance();
        try {
            CryptoSuite cryptoSuite = CryptoSuiteFactory.getDefault().getCryptoSuite();
            client.setCryptoSuite(cryptoSuite);
        } catch (CryptoException | InvalidArgumentException | ClassNotFoundException | IllegalAccessException
                | InstantiationException | NoSuchMethodException | InvocationTargetException e) {
            throw new RuntimeException("Client init failed: " + e.getMessage());
        }
        client.setUserContext(user);
        // Load the channel
        try {
            channel = client.loadChannelFromConfig(channelName, networkConfig);
            channel.initialize();
        } catch (InvalidArgumentException | NetworkConfigurationException | TransactionException e) {
            throw new RuntimeException("Channel load failed: " + e.getMessage());
        }
    }

    public Collection<Peer> getAllPeers() {
        return channel.getPeers();
    }

    public Collection<Peer> getPeersForOrg(String mspId) throws InvalidArgumentException {
        return channel.getPeersForOrganization(mspId);
    }

    public void setUser(User user) {
        client.setUserContext(user);
    }

    public void setChaincode(String chaincodeName) {
        ccid = ChaincodeID.newBuilder().setName(chaincodeName).build();
    }

    public CompletableFuture<BlockEvent.TransactionEvent> sendTransaction(String function, String[] arguments)
            throws InvalidArgumentException, ProposalException {
        TransactionProposalRequest tpr = client.newTransactionProposalRequest();
        tpr.setChaincodeID(ccid);
        tpr.setFcn(function);
        tpr.setArgs(arguments);
        tpr.setProposalWaitTime(proposalWaitTime);
        Collection<ProposalResponse> responses = channel.sendTransactionProposal(tpr);
        return channel.sendTransaction(responses);
    }

    public Collection<ProposalResponse> queryPeers(Collection<Peer> peers, String function, String[] arguments) throws ProposalException, InvalidArgumentException {
        QueryByChaincodeRequest queryRequest = client.newQueryProposalRequest();
        queryRequest.setChaincodeID(ccid);
        queryRequest.setFcn(function);
        if (arguments != null)
            queryRequest.setArgs(arguments);
        return channel.queryByChaincode(queryRequest, peers);
    }
}
