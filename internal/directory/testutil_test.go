/* SPDX-License-Identifier: Apache-2.0
 *
 * Copyright 2023 Damian Peckett <damian@pecke.tt>.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package directory_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

func generateCertificates(certsDir string) error {
	caKeyPEM, caCertPEM, err := generateSelfSignedCA()
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(certsDir, "ca.crt"), caCertPEM, 0o644); err != nil {
		return err
	}

	serverKeyPEM, serverCertPEM, err := generateServerCertificate(caKeyPEM, caCertPEM)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(certsDir, "tls.crt"), serverCertPEM, 0o644); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(certsDir, "tls.key"), serverKeyPEM, 0o644); err != nil {
		return err
	}

	return nil
}

func generateSelfSignedCA() (caKeyPEM, caCertPEM []byte, err error) {
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "Test CA",
			Organization: []string{"Acme Widgets Inc."},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertBytes, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	caKeyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey)})
	caCertPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertBytes})

	return caKeyPEM, caCertPEM, nil
}

func generateServerCertificate(caKeyPEM, caCertPEM []byte) (certPEM, keyPEM []byte, err error) {
	caKeyBlock, _ := pem.Decode(caKeyPEM)
	caPrivKey, err := x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	caCertBlock, _ := pem.Decode(caCertPEM)
	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName:   "Test Cert",
			Organization: []string{"Acme Widgets Inc."},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{"localhost"},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &certTemplate, caCert, &privKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privKey)})
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})

	return keyPEM, certPEM, nil
}
