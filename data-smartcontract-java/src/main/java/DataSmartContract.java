import org.hyperledger.fabric.contract.Context;
import org.hyperledger.fabric.contract.ContractInterface;
import org.hyperledger.fabric.contract.annotation.Contract;
import org.hyperledger.fabric.contract.annotation.Default;
import org.hyperledger.fabric.contract.annotation.Transaction;
import org.hyperledger.fabric.shim.ChaincodeException;
import org.hyperledger.fabric.shim.ChaincodeStub;

@Contract
@Default
public class DataSmartContract implements ContractInterface {
    private enum DataSmartContractErrors {
        KEY_NOT_FOUND
    }

    @Transaction(intent = Transaction.TYPE.SUBMIT)
    public void storeValue(final Context ctx, final String key, final String value) {
        ChaincodeStub stub = ctx.getStub();
        stub.putStringState(key, value);
    }
    @Transaction(intent = Transaction.TYPE.SUBMIT)
    public void deleteValue(final Context ctx, final String key) {
        ChaincodeStub stub = ctx.getStub();
        String value = stub.getStringState(key);
        if (value == null || value.isEmpty()) {
            String errorMessage = String.format("Key %s does not exist", key);
            throw new ChaincodeException(errorMessage, DataSmartContractErrors.KEY_NOT_FOUND.toString());
        }
        stub.delState(key);
    }
    @Transaction(intent = Transaction.TYPE.EVALUATE)
    public String readValue(final Context ctx, final String key) {
        ChaincodeStub stub = ctx.getStub();
        String value = stub.getStringState(key);
        if (value == null || value.isEmpty()) {
            String errorMessage = String.format("Key %s does not exist", key);
            throw new ChaincodeException(errorMessage, DataSmartContractErrors.KEY_NOT_FOUND.toString());
        }
        return value;
    }
}
