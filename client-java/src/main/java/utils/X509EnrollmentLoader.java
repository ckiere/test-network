package utils;

import org.apache.commons.codec.binary.Base64;
import org.hyperledger.fabric.sdk.Enrollment;
import org.hyperledger.fabric.sdk.identity.IdemixEnrollment;
import org.hyperledger.fabric.sdk.identity.X509Enrollment;

import java.io.File;
import java.io.IOException;
import java.nio.charset.Charset;
import java.nio.file.Files;
import java.security.KeyFactory;
import java.security.NoSuchAlgorithmException;
import java.security.PrivateKey;
import java.security.interfaces.ECPrivateKey;
import java.security.spec.InvalidKeySpecException;
import java.security.spec.PKCS8EncodedKeySpec;

public class X509EnrollmentLoader {
    public static X509Enrollment loadFromFile(File privateKeyFile, File certFile) throws InvalidKeySpecException, NoSuchAlgorithmException, IOException {
        PrivateKey pk = readPrivateKey(readFile(privateKeyFile));
        String cert = readFile(certFile);
        return new X509Enrollment(pk, cert);
    }

    // Adapted from https://www.baeldung.com/java-read-pem-file-keys
    private static ECPrivateKey readPrivateKey(String key) throws IOException, InvalidKeySpecException, NoSuchAlgorithmException {
        String privateKeyPEM = key
                .replace("-----BEGIN PRIVATE KEY-----", "")
                .replaceAll("\r", "")
                .replaceAll("\n", "")
                .replace("-----END PRIVATE KEY-----", "");
        byte[] encoded = Base64.decodeBase64(privateKeyPEM);
        KeyFactory keyFactory = KeyFactory.getInstance("EC");
        PKCS8EncodedKeySpec keySpec = new PKCS8EncodedKeySpec(encoded);
        return (ECPrivateKey) keyFactory.generatePrivate(keySpec);
    }

    private static String readFile(File file) throws IOException {
        return Files.readString(file.toPath());
    }
}
