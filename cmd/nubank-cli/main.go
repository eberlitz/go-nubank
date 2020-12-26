package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lucsky/cuid"
	"golang.org/x/term"

	"github.com/eberlitz/go-nubank/pkg/client"
)

func main() {
	c, err := client.New()
	if err != nil {
		log.Fatalf("failed to create Nubank Client: %+v", err)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your CPF(Numbers only): ")
	cpf, _ := reader.ReadString('\n')
	cpf = strings.TrimSuffix(cpf, "\n")

	fmt.Printf("Now, please enter your password (Used on the app/website): ")
	password, _ := term.ReadPassword(0)

	fmt.Println("\n\nRequesting e-mail code ...")

	ctx := context.Background()
	deviceID := newDeviceID()
	codeRequest, err := c.RequestCode(ctx, cpf, string(password), deviceID)
	if err != nil {
		log.Fatalf("failed to request code for certificate generation: %+v", err)
	}
	fmt.Print("Type code received by e-mail: ")
	code, _ := reader.ReadString('\n')
	code = strings.TrimSuffix(code, "\n")

	key, cert, err := codeRequest.ExchangeCerts(ctx, code)
	if err != nil {
		log.Fatalf("failed to exchange code for certificate: %+v", err)
	}

	certFilePath := "./cert.pem"
	err = saveString(certFilePath, cert)
	if err != nil {
		log.Fatalln(err)
	}
	keyFilePath := "./key.pem"
	err = saveString(keyFilePath, key)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("Certificate generated at %q and %q\n", certFilePath, keyFilePath)
}

// newDeviceID returns a 12 digit random string to use as a device id.
func newDeviceID() string {
	deviceID := cuid.New()
	runes := []rune(deviceID)
	deviceID = string(runes[13:25])
	return deviceID
}

func saveString(filepath, content string) error {
	certOut, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing: %w", filepath, err)
	}
	if _, err := certOut.WriteString(content); err != nil {
		return fmt.Errorf("failed to write data to %s: %w", filepath, err)
	}
	if err := certOut.Sync(); err != nil {
		return fmt.Errorf("error syncing %s: %w", filepath, err)
	}
	if err := certOut.Close(); err != nil {
		return fmt.Errorf("error closing %s: %w", filepath, err)
	}
	return nil
}
