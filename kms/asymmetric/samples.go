// Copyright 2018 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Samples for asymmetric keys feature of Cloud Key Management Service: https://cloud.google.com/kms/
package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"

	"golang.org/x/net/context"
	"google.golang.org/api/cloudkms/v1"
)

// [START kms_get_asymmetric_public]

// getAsymmetricPublicKey retrieves the public key from a saved asymmetric key pair on KMS.
func getAsymmetricPublicKey(ctx context.Context, client *cloudkms.Service, keyPath string) (interface{}, error) {
	response, err := client.Projects.Locations.KeyRings.CryptoKeys.CryptoKeyVersions.
		GetPublicKey(keyPath).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch public key: %+v", err)
	}
	keyBytes := []byte(response.Pem)
	block, _ := pem.Decode(keyBytes)
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %+v", err)
	}
	return publicKey, nil
}

// [END kms_get_asymmetric_public]

// [START kms_decrypt_rsa]

// decryptRSA will attempt to decrypt a given ciphertext with saved a RSA key.
func decryptRSA(ctx context.Context, client *cloudkms.Service, ciphertext, keyPath string) (string, error) {
	decryptRequest := &cloudkms.AsymmetricDecryptRequest{
		Ciphertext: ciphertext,
	}
	response, err := client.Projects.Locations.KeyRings.CryptoKeys.CryptoKeyVersions.
		AsymmetricDecrypt(keyPath, decryptRequest).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("decryption request failed: %+v", err)
	}
	message, err := base64.StdEncoding.DecodeString(response.Plaintext)
	if err != nil {
		return "", fmt.Errorf("failed to decode decryted string: %+v", err)

	}
	return string(message), nil
}

// [END kms_decrypt_rsa]

// [START kms_encrypt_rsa]

// encryptRSA creates a ciphertext from a plain message using a RSA public key saved at the specified keyPath.
func encryptRSA(ctx context.Context, client *cloudkms.Service, message, keyPath string) (string, error) {
	abstractKey, err := getAsymmetricPublicKey(ctx, client, keyPath)
	if err != nil {
		return "", err
	}

	// Perform type assertion to get the RSA key.
	rsaKey := abstractKey.(*rsa.PublicKey)

	ciphertextBytes, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaKey, []byte(message), nil)
	if err != nil {
		return "", fmt.Errorf("encryption failed: %+v", err)
	}
	return base64.StdEncoding.EncodeToString(ciphertextBytes), nil
}

// [END kms_encrypt_rsa]

// [START kms_sign_asymmetric]

// signAsymmetric will sign a plaintext message using a saved asymmetric private key.
func signAsymmetric(ctx context.Context, client *cloudkms.Service, message, keyPath string) (string, error) {
	// Find the hash of the plaintext message.
	digest := sha256.New()
	digest.Write([]byte(message))
	digestStr := base64.StdEncoding.EncodeToString(digest.Sum(nil))

	asymmetricSignRequest := &cloudkms.AsymmetricSignRequest{
		Digest: &cloudkms.Digest{
			Sha256: digestStr,
		},
	}

	response, err := client.Projects.Locations.KeyRings.CryptoKeys.CryptoKeyVersions.
		AsymmetricSign(keyPath, asymmetricSignRequest).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("asymmetric sign request failed: %+v", err)

	}

	return response.Signature, nil
}

// [END kms_sign_asymmetric]

// [START kms_verify_signature_rsa]

// verifySignatureRSA will verify that an 'RSA_SIGN_PSS_2048_SHA256' signature is valid for a given plaintext message.
func verifySignatureRSA(ctx context.Context, client *cloudkms.Service, signature, message, keyPath string) error {
	abstractKey, err := getAsymmetricPublicKey(ctx, client, keyPath)
	if err != nil {
		return err
	}
	// Perform type assertion to get the RSA key.
	rsaKey := abstractKey.(*rsa.PublicKey)
	decodedSignature, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature string: %+v", err)

	}
	digest := sha256.New()
	digest.Write([]byte(message))
	hash := digest.Sum(nil)

	pssOptions := rsa.PSSOptions{SaltLength: len(hash), Hash: crypto.SHA256}
	err = rsa.VerifyPSS(rsaKey, crypto.SHA256, hash, decodedSignature, &pssOptions)
	if err != nil {
		return fmt.Errorf("signature verification failed: %+v", err)
	}
	return nil
}

// [END kms_verify_signature_rsa]

// [START kms_verify_signature_ec]

// verifySignatureEC will verify that an 'EC_SIGN_P224_SHA256' signature is valid for a given plaintext message.
func verifySignatureEC(ctx context.Context, client *cloudkms.Service, signature, message, keyPath string) error {
	abstractKey, err := getAsymmetricPublicKey(ctx, client, keyPath)
	if err != nil {
		return err
	}
	// Perform type assertion to get the elliptic curve key.
	ecKey := abstractKey.(*ecdsa.PublicKey)
	decodedSignature, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature string: %+v", err)
	}
	var parsedSig struct{ R, S *big.Int }
	_, err = asn1.Unmarshal(decodedSignature, &parsedSig)
	if err != nil {
		return fmt.Errorf("failed to parse signature bytes: %+v", err)
	}

	digest := sha256.New()
	digest.Write([]byte(message))
	hash := digest.Sum(nil)

	if !ecdsa.Verify(ecKey, hash, parsedSig.R, parsedSig.S) {
		return errors.New("signature verification failed")
	}
	return nil
}

// [END kms_verify_signature_ec]
