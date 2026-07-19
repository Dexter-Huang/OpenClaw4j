package legacycrypto

import "testing"

func TestJavaAPIKeyEncryptorMatchesJavaAESCryptUtils(t *testing.T) {
	encryptor, err := NewJavaAPIKeyEncryptor()
	if err != nil {
		t.Fatalf("NewJavaAPIKeyEncryptor returned error: %v", err)
	}

	ciphertext, err := encryptor.Encrypt("sk-legacy-key")
	if err != nil {
		t.Fatalf("Encrypt returned error: %v", err)
	}
	if ciphertext != "vVMcuw93pr1wPLoeEE513Q==" {
		t.Fatalf("ciphertext did not match Java AES output: %q", ciphertext)
	}
	plain, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt returned error: %v", err)
	}
	if plain != "sk-legacy-key" {
		t.Fatalf("decrypted plaintext did not match: %q", plain)
	}
}

func TestJavaAPIKeyEncryptorPadsFullBlocks(t *testing.T) {
	encryptor, err := NewJavaAPIKeyEncryptor()
	if err != nil {
		t.Fatalf("NewJavaAPIKeyEncryptor returned error: %v", err)
	}

	ciphertext, err := encryptor.Encrypt("1234567890abcdef")
	if err != nil {
		t.Fatalf("Encrypt returned error: %v", err)
	}
	if ciphertext != "qVSzghsIK6Yn3gVcKNy43BzTCVtra1e8gzxG1UQIu+A=" {
		t.Fatalf("full-block padding did not match expected AES output: %q", ciphertext)
	}
}
