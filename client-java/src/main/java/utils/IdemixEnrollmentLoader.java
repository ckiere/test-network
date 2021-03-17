package utils;

import java.io.StringReader;
import java.security.PublicKey;
import java.security.interfaces.ECPublicKey;
import java.security.spec.X509EncodedKeySpec;
import java.util.Base64;

import org.apache.milagro.amcl.FP256BN.BIG;
import org.hyperledger.fabric.protos.idemix.Idemix;
import org.hyperledger.fabric.sdk.idemix.IdemixCredential;
import org.hyperledger.fabric.sdk.idemix.IdemixIssuerPublicKey;
import org.hyperledger.fabric.sdk.idemix.IdemixUtils;
import org.hyperledger.fabric.sdk.identity.IdemixEnrollment;
import org.hyperledger.fabric.sdk.identity.X509Enrollment;

import javax.json.Json;
import javax.json.JsonObject;
import javax.json.JsonReader;
import java.io.File;
import java.io.IOException;
import java.nio.file.Files;
import java.security.KeyFactory;
import java.security.NoSuchAlgorithmException;
import java.security.PrivateKey;
import java.security.interfaces.ECPrivateKey;
import java.security.spec.InvalidKeySpecException;
import java.security.spec.PKCS8EncodedKeySpec;
import static java.nio.charset.StandardCharsets.UTF_8;

public class IdemixEnrollmentLoader {
    public static IdemixEnrollment loadFromFile(File configFile, File ipkFile, File rpkFile, String mspId, int role)
            throws InvalidKeySpecException, NoSuchAlgorithmException, IOException {
        JsonObject config = readJsonFile(configFile);

        // Get issuer public key
        byte[] ipkBytes = readBinaryFile(ipkFile);
        Idemix.IssuerPublicKey ipkProto = Idemix.IssuerPublicKey.parseFrom(ipkBytes);
        IdemixIssuerPublicKey ipk = new IdemixIssuerPublicKey(ipkProto);

        // Get revocation public key
        PublicKey rpk = readPublicKey(readFile(rpkFile));

        // Get private key
        BIG sk = BIG.fromBytes(Base64.getDecoder().decode(config.getString("Sk")));

        // Deserialize idemix credential
        String credential = config.getString("Cred");
        byte[] credBytes = Base64.getDecoder().decode(credential.getBytes(UTF_8));
        Idemix.Credential credProto = Idemix.Credential.parseFrom(credBytes);
        IdemixCredential cred = new IdemixCredential(credProto);

        // Deserialize idemix cri (Credential Revocation Information)
        String criStr = config.getString("credential_revocation_information");
        byte[] criBytes = Base64.getDecoder().decode(criStr.getBytes(UTF_8));
        Idemix.CredentialRevocationInformation cri = Idemix.CredentialRevocationInformation.parseFrom(criBytes);


        String ou = config.getString("organizational_unit_identifier");

        return new IdemixEnrollment(ipk, rpk, mspId, sk, cred, cri, ou, role);
    }

    // Adapted from https://www.baeldung.com/java-read-pem-file-keys
    private static ECPublicKey readPublicKey(String key) throws IOException, InvalidKeySpecException, NoSuchAlgorithmException {
        String publicKeyPem = key
                .replace("-----BEGIN PUBLIC KEY-----", "")
                .replaceAll("\r", "")
                .replaceAll("\n", "")
                .replace("-----END PUBLIC KEY-----", "");
        byte[] encoded = Base64.getDecoder().decode(publicKeyPem.getBytes(UTF_8));
        KeyFactory keyFactory = KeyFactory.getInstance("EC");
        X509EncodedKeySpec keySpec = new X509EncodedKeySpec(encoded);
        return (ECPublicKey) keyFactory.generatePublic(keySpec);
    }

    private static String readFile(File file) throws IOException {
        return Files.readString(file.toPath());
    }

    private static byte[] readBinaryFile(File file) throws IOException {
        return Files.readAllBytes(file.toPath());
    }

    private static JsonObject readJsonFile(File file) throws IOException {
        String jsonString = readFile(file);
        JsonReader reader = Json.createReader(new StringReader(jsonString));
        return reader.readObject();
    }
}
