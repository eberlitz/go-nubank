package client

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type Code struct {
	cli         *client
	request     *certificateRequest
	codeReqData map[string]string
}

type certificateRequest struct {
	key1    *rsa.PrivateKey
	key2    *rsa.PrivateKey
	payload map[string]string
}

func (g *Code) ExchangeCerts(ctx context.Context, code string) (string, string, error) {
	if g.request.payload == nil || g.request.payload["encrypted-code"] == "" {
		return "", "", ErrNoEncryptedCode
	}
	g.request.payload["code"] = code

	req, err := g.cli.newJSONPostRequest(ctx, g.cli.appLinks["gen_certificate"], g.request.payload)
	if err != nil {
		return "", "", err
	}

	response, err := g.cli.doRequest(req)
	if err != nil {
		return "", "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("%w: Got %d, Want %d", ErrStatusCodeUnexpected, response.StatusCode, http.StatusOK)
	}

	var certResponse map[string]string
	err = json.NewDecoder(response.Body).Decode(&certResponse)
	if err != nil {
		return "", "", err
	}

	cert := certResponse["certificate"]

	privBytes, err := x509.MarshalPKCS8PrivateKey(g.request.key1)
	if err != nil {
		return "", "", fmt.Errorf("could not marshal private key: %w", err)
	}
	key := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

	return string(key), cert, nil
}

func generateKey() *rsa.PrivateKey {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate RSA key: %v", err)
	}
	return priv
}

func getPublicKeyPEMStr(key *rsa.PrivateKey) string {
	pubkeyBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		panic(err)
	}
	return string(pem.EncodeToMemory(
		&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubkeyBytes,
		},
	))
}

func parseAuthenticateHeader(d string) map[string]string {
	d = strings.TrimPrefix(d, "device-authorization ")
	result := map[string]string{}
	for _, kv := range strings.Split(d, ",") {
		kvs := strings.Split(kv, "=")
		if len(kvs) == 2 {
			result[strings.TrimSpace(kvs[0])] = strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(kvs[1]), "\""), "\"")
		}
	}
	return result
}
